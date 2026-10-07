package flow

import (
	"sync"
	"time"
)

type limitEntry struct {
	tokens float64
	at     time.Time
	rate   int
}
type ManagementLimiter struct {
	mu      sync.Mutex
	entries map[string]limitEntry
	cleaned time.Time
}

// Caller supplies the direct peer IP, never a forwarded header. The map is
// bounded even if many clients rotate addresses.
func (l *ManagementLimiter) Allow(key string, rate int, now time.Time) bool {
	if rate <= 0 {
		return true
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.entries == nil {
		l.entries = make(map[string]limitEntry)
	}
	if now.Sub(l.cleaned) >= time.Minute {
		for k, e := range l.entries {
			if now.Sub(e.at) > 2*time.Minute {
				delete(l.entries, k)
			}
		}
		l.cleaned = now
	}
	e, ok := l.entries[key]
	burst := float64(min(rate, 60))
	if !ok {
		if len(l.entries) >= 4096 {
			return false
		}
		e = limitEntry{burst, now, rate}
	}
	if e.rate != rate {
		e.tokens = min(e.tokens, burst)
		e.rate = rate
	}
	e.tokens = min(burst, e.tokens+max(0, now.Sub(e.at).Seconds())*float64(rate)/60)
	e.at = now
	allow := e.tokens >= 1
	if allow {
		e.tokens--
	}
	l.entries[key] = e
	return allow
}
