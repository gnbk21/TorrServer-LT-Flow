package torr

import (
	"net/http/httptest"
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
	tor.flowProgress(1, "test", second, 70<<20)
	tor.flowProgress(1, "test", first, 21<<20)
	tor.flowEnd(1, "test", first, FlowRangeTrace{BytesServed: 21 << 20})
	if s.PlaybackOffsetBytes != 70<<20 {
		t.Fatal("old response overwrote seek progress")
	}
	req.Header.Set("Range", "bytes=-1024")
	_, _, probe := tor.flowStart(1, file, "test", req)
	tor.flowProgress(1, "test", probe, file.Length)
	if s.PlaybackOffsetBytes != 70<<20 {
		t.Fatal("tail probe overwrote playback progress")
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
