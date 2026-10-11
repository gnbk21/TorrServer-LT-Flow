package flow

import (
	"sync"
	"testing"
	"time"
)

func TestStorageEvidenceThresholdAndOperationClock(t *testing.T) {
	var observations StorageObservations
	start := time.Unix(100, 0).UTC()
	observations.Record(false, time.Time{}, time.Second, 12, true)
	observations.Record(false, start, -time.Second, 12, true)
	observations.Record(false, start, StorageSlowOperation-time.Nanosecond, 12, false)
	observations.Record(true, start, StorageSlowOperation, 12, false)
	observations.Record(false, start, 0, -1, true)
	rows, dropped := observations.Drain()
	if len(rows) != 2 || dropped != 0 || rows[0].Stage != "CACHE_WRITE_SLOW" || rows[1].Stage != "CACHE_READ_FAILED" {
		t.Fatalf("incorrect threshold or error evidence: %+v", rows)
	}
	if !rows[0].Time.Equal(start.Add(StorageSlowOperation)) || rows[0].ElapsedMs != 50 || !rows[0].OperationMeasured || rows[1].Bytes != 0 {
		t.Fatal("operation completion clock or measured zero was lost")
	}
	if rows, dropped := observations.Drain(); len(rows) != 0 || dropped != 0 {
		t.Fatal("drain retained old events")
	}
}

func TestStorageEvidenceBoundedConcurrentProducers(t *testing.T) {
	var observations StorageObservations
	var producers sync.WaitGroup
	for i := 0; i < 8; i++ {
		producers.Add(1)
		go func() {
			defer producers.Done()
			for n := 0; n < 100; n++ {
				observations.Record(true, time.Now(), time.Millisecond, n, true)
			}
		}()
	}
	producers.Wait()
	rows, dropped := observations.Drain()
	if len(rows) != StorageObservationLimit || dropped != 800-StorageObservationLimit {
		t.Fatal("storage evidence exceeded its fixed bound or lost drop accounting")
	}
}

func TestDelayedStorageEvidenceUsesObservedClock(t *testing.T) {
	var timeline Timeline
	start := time.Unix(100, 0).UTC()
	timeline.Record(HistoryEvent{Type: "seek", Time: start.Add(time.Second)}, time.Second)
	timeline.Record(HistoryEvent{Type: "storage", Stage: "CACHE_WRITE_SLOW", Time: start.Add(200 * time.Millisecond), OperationMeasured: true, ElapsedMs: 50}, 200*time.Millisecond)
	timeline.AddDropped(3)
	rows := timeline.Snapshot()
	if rows.Dropped != 3 || rows.Events[0].Type != "storage" || !rows.Events[0].Time.Equal(start.Add(200*time.Millisecond)) || *rows.Events[0].OperationMs != 50 {
		t.Fatal("late delivery changed the measured storage event clock")
	}
}
