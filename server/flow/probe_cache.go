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
	lease         uint64
	result        ProbeResult
	ready         bool
	expires, used time.Time
}
type ProbeCache struct {
	mu       sync.Mutex
	entries  map[ProbeKey]probeEntry
	sequence uint64
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
func (c *ProbeCache) Begin(key ProbeKey, now time.Time) uint64 {
	if key.Hash == "" || key.Size <= 0 {
		return 0
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.entries[key]; ok && now.Before(e.expires) {
		return 0
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
	c.sequence++
	if c.sequence == 0 {
		c.sequence++
	}
	c.entries[key] = probeEntry{lease: c.sequence, expires: now.Add(35 * time.Second), used: now}
	return c.sequence
}

func (c *ProbeCache) Finish(key ProbeKey, lease uint64, result ProbeResult, success bool, now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	// An evicted lease cannot repopulate the map and exceed its capacity.
	if entry, ok := c.entries[key]; !ok || lease == 0 || entry.lease != lease {
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
