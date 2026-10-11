package flow

import (
	"sync"
	"time"
)

const StorageObservationLimit = 16
const StorageSlowOperation = 50 * time.Millisecond

// Per-cache operation evidence, not process-wide disk attribution. Capture is
// bounded and cannot contain paths, free-text errors or media identities.
type StorageObservations struct {
	mu      sync.Mutex
	events  [StorageObservationLimit]HistoryEvent
	count   int
	dropped uint64
}

func (s *StorageObservations) Record(write bool, started time.Time, elapsed time.Duration, bytes int, failed bool) {
	if started.IsZero() || elapsed < 0 || (!failed && elapsed < StorageSlowOperation) {
		return
	}
	stage := "CACHE_READ_SLOW"
	if write {
		stage = "CACHE_WRITE_SLOW"
	}
	if failed {
		if write {
			stage = "CACHE_WRITE_FAILED"
		} else {
			stage = "CACHE_READ_FAILED"
		}
	}
	e := HistoryEvent{Type: "storage", Stage: stage, Time: started.Add(elapsed).UTC(),
		OperationMeasured: true, ElapsedMs: elapsed.Milliseconds(), Bytes: int64(max(0, bytes))}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.count == len(s.events) {
		copy(s.events[:], s.events[1:])
		s.count--
		s.dropped++
	}
	s.events[s.count] = e
	s.count++
}

func (s *StorageObservations) Drain() ([]HistoryEvent, uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.count == 0 && s.dropped == 0 {
		return nil, 0
	}
	events, dropped := append([]HistoryEvent(nil), s.events[:s.count]...), s.dropped
	clear(s.events[:])
	s.count, s.dropped = 0, 0
	return events, dropped
}
