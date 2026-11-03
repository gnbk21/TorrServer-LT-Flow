package torr

import (
	"net/http"
	"server/flow"
	"server/lt"
	"server/settings"
	"server/torr/storage/torrstor"
	"testing"
	"time"
)

func TestRecentPurposeExcludesHeaderIndexAndInternalProbesWithoutMetrics(t *testing.T) {
	old := settings.BTsets()
	cfg := settings.NewDefaultConfig()
	cfg.Flow.MetricsEnabled = false
	cfg.Flow.RangeClassification = false
	settings.StoreBTsets(cfg)
	t.Cleanup(func() { settings.StoreBTsets(old) })
	for _, tc := range []struct {
		method, group, header string
		size                  int64
		want                  bool
	}{
		{http.MethodGet, "client", "bytes=0-1023", 100 << 20, false},
		{http.MethodGet, "client", "bytes=-65536", 100 << 20, false},
		{http.MethodHead, "client", "", 100 << 20, false},
		{http.MethodGet, torrstor.ProbeReaderGroup, "bytes=0-", 100 << 20, false},
		{http.MethodGet, "client", "bytes=0-", 100 << 20, true},
		{http.MethodGet, "client", "bytes=0-9999999", 1 << 20, true},
		{http.MethodGet, "client", "", 100 << 20, true},
	} {
		if got := recentPlaybackRequest(tc.method, tc.group, flow.ParseRangeHint(tc.header, tc.size), tc.size); got != tc.want {
			t.Fatalf("%+v: %v", tc, got)
		}
	}
}

func TestRecentQueueKeepsConcurrentFilesAndExcludesProbes(t *testing.T) {
	old := settings.BTsets()
	settings.StoreBTsets(settings.NewDefaultConfig())
	t.Cleanup(func() { settings.StoreBTsets(old) })
	tor := &Torrent{TorrentSpec: &TorrentSpec{InfoHash: Hash{1}}}
	tor.lh.Store(&lt.Torrent{})
	one, two := &File{Index: 0, Path: "one.mkv", Length: 100}, &File{Index: 1, Path: "two.mp4", Length: 100}
	tor.filesSnapshot.Store(&torrentFiles{sorted: []*File{one, two}, byID: map[int]*File{1: one, 2: two}})
	tor.queueRecentOpen(1, torrstor.ProbeReaderGroup, "probe")
	if tor.recentOpen.Load() != nil {
		t.Fatal("probe became Recent")
	}
	tor.queueRecentOpen(1, "client", "session-a")
	tor.queueRecentOpen(2, "client", "session-b")
	tor.queueRecentOpen(1, "client", "session-a")
	queue := tor.recentOpen.Load()
	if queue == nil || len(queue.events) != 2 {
		t.Fatal("concurrent opens lost or duplicated", queue)
	}
}
func TestRecentGroupsWithoutMetricsKeepActiveAndWarmSessions(t *testing.T) {
	old := settings.BTsets()
	cfg := settings.NewDefaultConfig()
	cfg.Flow.MetricsEnabled = false
	settings.StoreBTsets(cfg)
	t.Cleanup(func() { settings.StoreBTsets(old) })
	tor := &Torrent{}
	first, closeFirst := tor.beginRecentDelivery(1, "client")
	overlapping, closeSecond := tor.beginRecentDelivery(1, "client")
	if first == "" || first != overlapping {
		t.Fatal("overlapping requests split sessions")
	}
	closeFirst()
	closeSecond()
	warm, closeWarm := tor.beginRecentDelivery(1, "client")
	closeWarm()
	if first != warm {
		t.Fatal("warm Range request created another open")
	}
	tor.recentDeliveryMu.Lock()
	for _, row := range tor.recentDeliveries {
		row.seen = time.Now().Add(-time.Hour)
	}
	tor.recentDeliveryMu.Unlock()
	later, closeLater := tor.beginRecentDelivery(1, "client")
	closeLater()
	if later == first {
		t.Fatal("a later playback could never enter Recent")
	}
	if settings.CurrentFlow().MetricsEnabled {
		t.Fatal("history enabled unrelated diagnostics")
	}
}
