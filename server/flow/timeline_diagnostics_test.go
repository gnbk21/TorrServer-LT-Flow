package flow

import (
	"testing"
	"time"
)

func TestPeerNoiseAndSnapshotsDoNotDisplaceTimeline(t *testing.T) {
	var timeline Timeline
	timeline.Record(HistoryEvent{Type: "startup", Stage: "READY"}, time.Second)
	for i := 0; i < 1000; i++ {
		for _, kind := range []string{"peer_disconnected", "peer_summary", "startup_sample", "playback_sample", "response"} {
			timeline.Record(HistoryEvent{Type: kind}, time.Second)
		}
	}
	timeline.Record(HistoryEvent{Type: "recovery", OperationMeasured: true, ElapsedMs: 2500}, 2*time.Second)
	if events := timeline.Snapshot().Events; len(events) != 2 || events[0].Stage != "READY" || events[1].Type != "recovery" {
		t.Fatal("noise displaced useful incidents")
	}
}
