package torr

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"server/lt"
	"server/settings"
)

func metadataTestTorrent(t *testing.T) (*Torrent, []byte) {
	t.Helper()
	old := settings.BTsets()
	settings.StoreBTsets(&settings.BTSets{Flow: settings.DefaultFlowSettings()})
	t.Cleanup(func() { settings.StoreBTsets(old) })
	payload := minimalTorrent()
	parsed, err := lt.ParseTorrentBytes(payload)
	if err != nil {
		t.Fatal(err)
	}
	s, err := lt.NewSession(lt.SessionConfig{"enable_dht": false, "enable_lsd": false, "enable_upnp": false, "enable_natpmp": false, "listen_interfaces": "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	h, err := s.AddTorrent(lt.AddTorrentParams{Link: "magnet:?xt=urn:btih:" + parsed.InfoHash, SavePath: t.TempDir(), Paused: true})
	if err != nil {
		t.Fatal(err)
	}
	tor := &Torrent{TorrentSpec: &TorrentSpec{InfoHash: NewHashFromHex(parsed.InfoHash)}, gotInfoCh: make(chan struct{}), closeCh: make(chan struct{}), flowAddedAt: time.Now()}
	tor.lh.Store(h)
	return tor, payload
}

func TestMetadataRecoveryCooldownDistinguishesMissingFromUnavailable(t *testing.T) {
	for _, unavailable := range []bool{false, true} {
		tor, _ := metadataTestTorrent(t)
		source := metadataSource{"https://missing-fixture.example/a", "SUPPLIED"}
		if unavailable {
			source.url = "https://unavailable-fixture.example/a"
		}
		var calls int
		fetch := func(context.Context, string) ([]byte, error) {
			calls++
			code := "SOURCE_MISSING"
			if unavailable {
				code = "SOURCE_UNAVAILABLE"
			}
			return nil, metadataSourceError{code: code}
		}
		tor.recoverMetadata(context.Background(), []metadataSource{source}, fetch)
		tor.recoverMetadata(context.Background(), []metadataSource{source}, fetch)
		if unavailable && calls != 1 || !unavailable && calls != 2 {
			t.Fatal("missing torrent opened a circuit, or outage was not cooled down", calls)
		}
	}
}

func TestMetadataRecoveryRetryIsBoundedAndDoesNotRestart(t *testing.T) {
	tor, _ := metadataTestTorrent(t)
	tor.TorrentSpec.MetadataSources = []string{"https://retry-fixture.example/a"}
	handle := tor.LTHandle()
	entered, exited := make(chan struct{}), make(chan struct{})
	fetch := func(ctx context.Context, _ string) ([]byte, error) {
		close(entered)
		defer close(exited)
		<-ctx.Done()
		return nil, ctx.Err()
	}
	if err := tor.launchMetadataRecovery(0, fetch); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		tor.mu.Lock()
		tor.metadataRecoveryCancel()
		tor.mu.Unlock()
		select {
		case <-exited:
		case <-time.After(time.Second):
			t.Error("retry cleanup did not finish")
		}
	})
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("retry never started")
	}
	if err := tor.launchMetadataRecovery(0, fetch); err == nil {
		t.Fatal("concurrent retry accepted")
	}
	tor.mu.Lock()
	tor.metadataRecoveryCancel()
	tor.mu.Unlock()
	select {
	case <-exited:
	case <-time.After(time.Second):
		t.Fatal("cancelled retry leaked")
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		tor.mu.Lock()
		running := tor.metadataRecoveryRunning
		tor.mu.Unlock()
		if !running {
			break
		}
		time.Sleep(time.Millisecond)
	}
	if err := tor.launchMetadataRecovery(0, fetch); err == nil {
		t.Fatal("immediate repeat bypassed rate limit")
	}
	if tor.LTHandle() != handle {
		t.Fatal("retry replaced the handle")
	}
}

func TestMetadataHedgeCancelsSlowLoserWithoutReplacingHandle(t *testing.T) {
	tor, payload := metadataTestTorrent(t)
	handle := tor.LTHandle()
	stopped := make(chan struct{})
	started := time.Now()
	tor.recoverMetadata(context.Background(), []metadataSource{{"https://hedge-slow.example/a", "SUPPLIED"}, {"https://hedge-good.example/a", "SUPPLIED"}}, func(ctx context.Context, raw string) ([]byte, error) {
		if raw == "https://hedge-slow.example/a" {
			<-ctx.Done()
			close(stopped)
			return nil, ctx.Err()
		}
		return payload, nil
	})
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("loser request leaked")
	}
	if elapsed := time.Since(started); elapsed < time.Second || elapsed > 3*time.Second {
		t.Fatal("hedge did not bound slow-source delay", elapsed)
	}
	if tor.LTHandle() != handle || tor.FlowStartup().MetadataRecovery.State != "METADATA_RECOVERED" {
		t.Fatal("failed verified hedge")
	}
}

func TestMetadataHedgeConsentWithdrawalCancelsOpenRequests(t *testing.T) {
	tor, _ := metadataTestTorrent(t)
	configured := settings.CloneSettings(settings.BTsets())
	configured.Flow.PublicMetadataCache = true
	settings.StoreBTsets(configured)
	started, stopped := make(chan struct{}), make(chan struct{})
	var calls atomic.Int32
	done := make(chan struct{})
	go func() {
		defer close(done)
		tor.recoverMetadata(context.Background(), []metadataSource{{"https://consent.example/a", "PUBLIC_CACHE"}, {"https://consent2.example/a", "PUBLIC_CACHE"}}, func(ctx context.Context, _ string) ([]byte, error) {
			calls.Add(1)
			close(started)
			<-ctx.Done()
			close(stopped)
			return nil, ctx.Err()
		})
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("request never started")
	}
	next := settings.CloneSettings(configured)
	next.Flow.PublicMetadataCache = false
	settings.StoreBTsets(next)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("consent withdrawal ignored")
	}
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("revoked request leaked")
	}
	if calls.Load() != 1 {
		t.Fatal("hedged after consent withdrawal")
	}
	if status := tor.FlowStartup().MetadataRecovery; status == nil || status.State != "METADATA_MISS" || status.Code != "POLICY_CHANGED" {
		t.Fatal("cancelled source still appears active", status)
	}
}
