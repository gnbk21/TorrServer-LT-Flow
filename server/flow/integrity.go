package flow

import (
	"sync"
	"time"
)

type IntegritySnapshot struct {
	Failures        uint64 `json:"failures"`
	RecentFailures  uint64 `json:"recent_failures"`
	WindowSeconds   int    `json:"window_seconds"`
	LastFailureAtMs int64  `json:"last_failure_at_ms"`
}

type IntegrityMeter struct {
	mu      sync.Mutex
	total   uint64
	last    int64
	buckets [60]struct {
		second int64
		count  uint64
	}
}

func (m *IntegrityMeter) Record(at time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	second := at.Unix()
	b := &m.buckets[uint64(second)%60]
	if b.second != second {
		b.second, b.count = second, 0
	}
	b.count++
	m.total++
	m.last = at.UnixMilli()
}

func (m *IntegrityMeter) Snapshot(now time.Time) IntegritySnapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := IntegritySnapshot{Failures: m.total, WindowSeconds: 60, LastFailureAtMs: m.last}
	for _, b := range m.buckets {
		if age := now.Unix() - b.second; age >= 0 && age < 60 {
			s.RecentFailures += b.count
		}
	}
	return s
}
