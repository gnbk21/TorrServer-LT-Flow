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
