package torr

import (
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"server/settings"
	"server/torr/state"
)

func TestFlowProgressFollowsLiveBodyAndIgnoresOldRequest(t *testing.T) {
	old := settings.BTsets()
	t.Cleanup(func() { settings.StoreBTsets(old) })
	f := settings.DefaultFlowSettings()
	f.Enabled = false // No native cache is needed for this metrics test.
	settings.StoreBTsets(&settings.BTSets{Flow: f})
	tor := &Torrent{}
	file := &File{Length: 256 << 20}
	req := httptest.NewRequest("GET", "/", nil)
	_, _, first := tor.flowStart(1, file, "test", req)
	tor.flowProgress(1, "test", first, 20<<20)
	s := tor.flowSessions["1/test"]
	if s.PlaybackOffsetBytes != 20<<20 {
		t.Fatal("open full response did not advance")
	}
	req.Header.Set("Range", "bytes=67108864-")
	_, _, second := tor.flowStart(1, file, "test", req)
	tor.flowBodyStart(1, "test", second, 64<<20)
	tor.flowBodyStart(1, "test", first, 0)
	if s.demandOffset != 64<<20 || s.PlaybackOffsetBytes != 20<<20 {
		t.Fatal("pending seek demand did not preserve delivered progress")
	}
	tor.flowProgress(1, "test", second, 70<<20)
	tor.flowProgress(1, "test", first, 21<<20)
	tor.flowEnd(1, "test", first, FlowRangeTrace{Method: "GET", BytesServed: 21 << 20})
	if s.PlaybackOffsetBytes != 70<<20 {
		t.Fatal("old response overwrote seek progress")
	}
	req.Header.Set("Range", "bytes=-1024")
	_, _, probe := tor.flowStart(1, file, "test", req)
	tor.flowBodyStart(1, "test", probe, file.Length-1024)
	tor.flowProgress(1, "test", probe, file.Length)
	if s.PlaybackOffsetBytes != 70<<20 || s.demandOffset != 70<<20 {
		t.Fatal("tail probe overwrote playback progress")
	}
}

func TestFlowHeadDoesNotBecomePlaybackAndErrorIsNotSeekRecovery(t *testing.T) {
	old := settings.BTsets()
	t.Cleanup(func() { settings.StoreBTsets(old) })
	f := settings.DefaultFlowSettings()
	f.Enabled = false
	settings.StoreBTsets(&settings.BTSets{Flow: f})
	tor := &Torrent{}
	file := &File{Length: 256 << 20}
	req := httptest.NewRequest("HEAD", "/", nil)
	_, _, seq := tor.flowStart(1, file, "test", req)
	s := tor.flowSessions["1/test"]
	if s.ActiveReaders != 0 || s.State == "PLAYING" {
		t.Fatal("HEAD became playback")
	}
	tor.flowEnd(1, "test", seq, FlowRangeTrace{Method: "HEAD", Status: 200})
	s.lastSeekSeq = seq
	tor.flowEnd(1, "test", seq, FlowRangeTrace{Method: "GET", Status: 416, BytesServed: 20, TTFBMs: 15})
	if s.SeekRecoveryMs != 0 {
		t.Fatal("error body became seek recovery")
	}
}

func TestWarmTimeoutUsesFlowSetting(t *testing.T) {
	old := settings.BTsets()
	t.Cleanup(func() { settings.StoreBTsets(old) })
	f := settings.DefaultFlowSettings()
	f.WarmSessionTimeoutSec = 900
	settings.StoreBTsets(&settings.BTSets{TorrentDisconnectTimeout: 30, Flow: f})
	if got := torrentExpireTimeout(); got != 900*time.Second {
		t.Fatalf("Flow timeout = %v", got)
	}
	f.Enabled = false
	if got := torrentExpireTimeout(); got != 30*time.Second {
		t.Fatalf("upstream timeout = %v", got)
	}
}

func TestWarmExpiryIgnoresStatusPollExtension(t *testing.T) {
	old := settings.BTsets()
	t.Cleanup(func() { settings.StoreBTsets(old) })
	settings.StoreBTsets(&settings.BTSets{Flow: settings.DefaultFlowSettings()})
	now := time.Now()
	tor := &Torrent{Stat: state.TorrentWorking, warmIdleSince: now.Add(-601 * time.Second), expiredTime: now.Add(time.Hour)}
	if !tor.expired(now) {
		t.Fatal("warm session should expire despite a later status-poll deadline")
	}
	tor.warmIdleSince = now.Add(-599 * time.Second)
	if tor.expired(now) {
		t.Fatal("warm session expired too soon")
	}
}

func TestStreamAdmissionAndIdleExpiryCannotBothWin(t *testing.T) {
	for i := 0; i < 200; i++ {
		tor := &Torrent{Stat: state.TorrentWorking, expiredTime: time.Now().Add(-time.Hour)}
		start := make(chan struct{})
		var workers sync.WaitGroup
		workers.Add(2)
		var admitted, claimed bool
		go func() { defer workers.Done(); <-start; admitted = tor.beginStream() }()
		go func() { defer workers.Done(); <-start; claimed = tor.claimExpiration(time.Now()) }()
		close(start)
		workers.Wait()
		if admitted == claimed {
			t.Fatal("admission and removal were not mutually exclusive")
		}
		if admitted {
			if tor.expired(time.Now()) {
				t.Fatal("in-flight HTTP setup expired before a reader existed")
			}
			tor.endStream()
			if !tor.claimExpiration(time.Now()) {
				t.Fatal("released request kept idle torrent alive")
			}
		}
		if tor.beginStream() {
			t.Fatal("closing torrent accepted a new stream")
		}
	}
}
