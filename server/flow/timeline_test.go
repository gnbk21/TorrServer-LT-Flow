package flow

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestTimelinePreservesStartupAndBoundsRecentTail(t *testing.T) {
	var timeline Timeline
	for i := 0; i < 1000; i++ {
		timeline.Record(HistoryEvent{Type: "first_byte", File: i, ElapsedMs: 5}, time.Duration(i)*time.Millisecond)
	}
	out := timeline.Snapshot()
	if len(out.Events) != 128 || out.Dropped != 872 || out.Events[0].File != 0 || out.Events[31].File != 31 || out.Events[32].File != 904 || out.Events[127].File != 999 {
		t.Fatal("timeline retention failed")
	}
	before := fmt.Sprint(out)
	timeline.Record(HistoryEvent{Type: "https://secret.example"}, time.Second)
	if fmt.Sprint(timeline.Snapshot()) != before {
		t.Fatal("arbitrary string recorded")
	}
	out.Events[0].File = -1
	if timeline.Snapshot().Events[0].File != 0 {
		t.Fatal("snapshot aliases internal state")
	}
}
func TestTimelineConcurrentProducers(t *testing.T) {
	var timeline Timeline
	var workers sync.WaitGroup
	for i := 0; i < 8; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for n := 0; n < 500; n++ {
				timeline.Record(HistoryEvent{Type: "metadata"}, time.Second)
				timeline.Snapshot()
			}
		}()
	}
	workers.Wait()
	if len(timeline.Snapshot().Events) != TimelineEvents {
		t.Fatal("incorrect event bound")
	}
}
