package flow

import (
	"math"
	"sync"
	"time"
)

// ProbeKey includes immutable torrent/file identity. Only media metadata is
// cached here; piece bytes remain exclusively in the existing media cache.
type ProbeKey struct {
	Hash  string
	Index int
	Size  int64
	Path  string
}
type ProbeResult struct {
	BitRate  string
	Duration float64
}
type probeEntry struct {
	result        ProbeResult
	ready         bool
	expires, used time.Time
}
type ProbeCache struct {
	mu      sync.Mutex
	entries map[ProbeKey]probeEntry
}

const probeCacheCapacity = 128

func (c *ProbeCache) Get(key ProbeKey, now time.Time) (ProbeResult, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[key]
	if !ok || !e.ready || !now.Before(e.expires) {
		return ProbeResult{}, false
	}
	e.used = now
	c.entries[key] = e
	return e.result, true
}

// Begin reserves one bounded probe lease. Failures back off for a minute;
// abandoned leases expire after 35 seconds, beyond the 30-second probe timeout.
func (c *ProbeCache) Begin(key ProbeKey, now time.Time) bool {
	if key.Hash == "" || key.Size <= 0 {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.entries[key]; ok && now.Before(e.expires) {
		return false
	}
	if c.entries == nil {
		c.entries = make(map[ProbeKey]probeEntry)
	}
	if len(c.entries) >= probeCacheCapacity {
		var oldest ProbeKey
		var used time.Time
		for k, e := range c.entries {
			if used.IsZero() || e.used.Before(used) {
				oldest, used = k, e.used
			}
		}
		delete(c.entries, oldest)
	}
	c.entries[key] = probeEntry{expires: now.Add(35 * time.Second), used: now}
	return true
}

func (c *ProbeCache) Finish(key ProbeKey, result ProbeResult, success bool, now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	// An evicted lease cannot repopulate the map and exceed its capacity.
	if _, ok := c.entries[key]; !ok {
		return
	}
	success = success && result.Duration >= 0 && !math.IsNaN(result.Duration) && !math.IsInf(result.Duration, 0) && (result.Duration > 0 || result.BitRate != "")
	ttl := time.Minute
	if success {
		ttl = 24 * time.Hour
	} else {
		result = ProbeResult{}
	}
	c.entries[key] = probeEntry{result: result, ready: success, used: now, expires: now.Add(ttl)}
}
