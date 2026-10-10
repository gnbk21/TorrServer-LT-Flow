package flow

import (
	"sync"
	"time"
)

const TimelineEvents = 128
const TimelineInitialEvents = 32

type TimelineEvent struct {
	Time        time.Time `json:"time"`
	ElapsedMs   int64     `json:"elapsed_ms"`
	OperationMs int64     `json:"operation_ms"`
	Type        string    `json:"type"`
	Stage       string    `json:"stage,omitempty"`
	File        int       `json:"file,omitempty"`
	Bytes       int64     `json:"bytes,omitempty"`
}
type TimelineSnapshot struct {
	Events  []TimelineEvent `json:"events"`
	Dropped uint64          `json:"dropped"`
	Source  string          `json:"source"`
}

// Preserve startup evidence and a bounded recent tail. Numeric, enum-only
// events can be exported without media identities, peer addresses or secrets.
type Timeline struct {
	mu      sync.Mutex
	events  []TimelineEvent
	dropped uint64
}

func (t *Timeline) Record(e HistoryEvent, elapsed time.Duration) {
	if !validHistoryEvent(e) || e.Type == "sparse" || e.Type == "dht" || elapsed < 0 {
		return
	}
	row := TimelineEvent{Time: time.Now().UTC(), ElapsedMs: elapsed.Milliseconds(), OperationMs: e.ElapsedMs, Type: e.Type, Stage: e.Stage, File: e.File, Bytes: e.Bytes}
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(t.events) == TimelineEvents {
		copy(t.events[TimelineInitialEvents:], t.events[TimelineInitialEvents+1:])
		t.events[len(t.events)-1] = row
		t.dropped++
	} else {
		t.events = append(t.events, row)
	}
}
func (t *Timeline) Snapshot() TimelineSnapshot {
	t.mu.Lock()
	defer t.mu.Unlock()
	return TimelineSnapshot{Events: append([]TimelineEvent{}, t.events...), Dropped: t.dropped, Source: "server"}
}
