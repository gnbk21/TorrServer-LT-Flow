package flow

import (
	"sync"
	"time"
)

// SourceHealth retains only hashed service/network-policy keys, never URLs or
// credentials. A torrent miss is not a service outage and must not open a circuit.
type SourceHealth struct {
	mu      sync.Mutex
	entries map[string]sourceHealthEntry
}
type sourceHealthEntry struct {
	until, seen time.Time
	failures    int
}

func (h *SourceHealth) Allowed(key string, now time.Time) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return !now.Before(h.entries[key].until)
}

func (h *SourceHealth) Observe(key string, unavailable bool, now time.Time) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if !unavailable {
		delete(h.entries, key)
		return
	}
	if h.entries == nil {
		h.entries = make(map[string]sourceHealthEntry)
	}
	if _, ok := h.entries[key]; !ok && len(h.entries) >= 64 {
		var oldest string
		var seen time.Time
		for k, e := range h.entries {
			if oldest == "" || e.seen.Before(seen) {
				oldest, seen = k, e.seen
			}
		}
		delete(h.entries, oldest)
	}
	e := h.entries[key]
	if now.Sub(e.seen) > 5*time.Minute {
		e.failures = 0
	}
	e.failures = min(3, e.failures+1)
	e.seen, e.until = now, now.Add(time.Duration(15<<(e.failures-1))*time.Second)
	h.entries[key] = e
}
