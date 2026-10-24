package flow

import (
	"sort"
	"sync"
	"time"
)

const TimelineEvents = 128
const TimelineInitialEvents = 32

type TimelineEvent struct {
	Time        time.Time `json:"time"`
	ElapsedMs   int64     `json:"elapsed_ms"`
	OperationMs *int64    `json:"operation_ms"`
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
	if !validHistoryEvent(e) || e.Type == "sparse" || e.Type == "stream_sample" || e.Type == "dht" || e.Type == "peer_disconnected" || e.Type == "peer_summary" || e.Type == "playback_sample" || e.Type == "startup_sample" || e.Type == "response" || elapsed < 0 {
		return
	}
	observed := e.Time
	if observed.IsZero() {
		observed = time.Now().UTC()
	}
	row := TimelineEvent{Time: observed, ElapsedMs: elapsed.Milliseconds(), Type: e.Type, Stage: e.Stage, File: e.File, Bytes: e.Bytes}
	if e.OperationMeasured && e.ElapsedMs >= 0 {
		duration := e.ElapsedMs
		row.OperationMs = &duration
	}
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
func (t *Timeline) AddDropped(count uint64) {
	t.mu.Lock()
	t.dropped += count
	t.mu.Unlock()
}
func (t *Timeline) Snapshot() TimelineSnapshot {
	t.mu.Lock()
	defer t.mu.Unlock()
	events := append([]TimelineEvent{}, t.events...)
	for i := range events {
		if events[i].OperationMs != nil {
			value := *events[i].OperationMs
			events[i].OperationMs = &value
		}
	}
	sort.SliceStable(events, func(i, j int) bool { return events[i].ElapsedMs < events[j].ElapsedMs })
	return TimelineSnapshot{Events: events, Dropped: t.dropped, Source: "server"}
}
