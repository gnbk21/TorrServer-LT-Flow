package flow

import (
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// Counters is a bounded per-torrent measurement set. The wait ring records
// recent experience without retaining one entry per piece read indefinitely.
type Counters struct {
	hitBytes    atomic.Int64
	missBytes   atomic.Int64
	waitCount   atomic.Int64
	waitTotalNs atomic.Int64
	waitsMu     sync.Mutex
	waits       [256]int64
	waitsAt     [256]int64
	waitsUsed   int
	waitsNext   int
	recent      [60]counterBucket
}

type counterBucket struct{ second, hit, miss, waits, stalls int64 }

type CounterSnapshot struct {
	CacheHitBytes          int64   `json:"cache_hit_bytes"`
	CacheMissBytes         int64   `json:"cache_miss_bytes"`
	PieceWaitCount         int64   `json:"piece_wait_count"`
	PieceWaitDurationMs    float64 `json:"piece_wait_duration_ms"`
	PieceWaitP50Ms         float64 `json:"piece_wait_p50_ms"`
	PieceWaitP95Ms         float64 `json:"piece_wait_p95_ms"`
	PieceWaitP99Ms         float64 `json:"piece_wait_p99_ms"`
	RecentWindowSeconds    int     `json:"recent_window_seconds"`
	RecentCacheHitBytes    int64   `json:"recent_cache_hit_bytes"`
	RecentCacheMissBytes   int64   `json:"recent_cache_miss_bytes"`
	RecentPieceWaitCount   int64   `json:"recent_piece_wait_count"`
	RecentPieceWaitP95Ms   float64 `json:"recent_piece_wait_p95_ms"`
	RecentServerReadStalls int64   `json:"recent_server_read_stalls"`
}

func (c *Counters) bucket(now time.Time) *counterBucket {
	second := now.Unix()
	b := &c.recent[second%int64(len(c.recent))]
	if b.second != second {
		*b = counterBucket{second: second}
	}
	return b
}

func (c *Counters) Hit(n int) {
	if n > 0 {
		c.hitBytes.Add(int64(n))
		c.waitsMu.Lock()
		c.bucket(time.Now()).hit += int64(n)
		c.waitsMu.Unlock()
	}
}
func (c *Counters) Miss(n int) {
	if n > 0 {
		c.missBytes.Add(int64(n))
		c.waitsMu.Lock()
		c.bucket(time.Now()).miss += int64(n)
		c.waitsMu.Unlock()
	}
}
func (c *Counters) Wait(d time.Duration) {
	if d < 0 {
		return
	}
	c.waitCount.Add(1)
	c.waitTotalNs.Add(int64(d))
	c.waitsMu.Lock()
	now := time.Now()
	b := c.bucket(now)
	b.waits++
	// A server read waited at least 500 ms for data; this is not a decoded-player stall.
	if d >= 500*time.Millisecond {
		b.stalls++
	}
	c.waits[c.waitsNext] = int64(d)
	c.waitsAt[c.waitsNext] = now.Unix()
	c.waitsNext = (c.waitsNext + 1) % len(c.waits)
	if c.waitsUsed < len(c.waits) {
		c.waitsUsed++
	}
	c.waitsMu.Unlock()
}

func (c *Counters) Snapshot() CounterSnapshot {
	return c.snapshotAt(time.Now())
}

func (c *Counters) snapshotAt(now time.Time) CounterSnapshot {
	s := CounterSnapshot{CacheHitBytes: c.hitBytes.Load(), CacheMissBytes: c.missBytes.Load(), PieceWaitCount: c.waitCount.Load()}
	s.RecentWindowSeconds = len(c.recent)
	if s.PieceWaitCount > 0 {
		s.PieceWaitDurationMs = float64(c.waitTotalNs.Load()) / float64(time.Millisecond) / float64(s.PieceWaitCount)
	}
	c.waitsMu.Lock()
	waits := append([]int64(nil), c.waits[:c.waitsUsed]...)
	recentWaits := make([]int64, 0, c.waitsUsed)
	for i := 0; i < c.waitsUsed; i++ {
		if age := now.Unix() - c.waitsAt[i]; age >= 0 && age < int64(len(c.recent)) {
			recentWaits = append(recentWaits, c.waits[i])
		}
	}
	for _, b := range c.recent {
		if age := now.Unix() - b.second; age >= 0 && age < int64(len(c.recent)) {
			s.RecentCacheHitBytes += b.hit
			s.RecentCacheMissBytes += b.miss
			s.RecentPieceWaitCount += b.waits
			s.RecentServerReadStalls += b.stalls
		}
	}
	c.waitsMu.Unlock()
	if len(recentWaits) > 0 {
		sort.Slice(recentWaits, func(i, j int) bool { return recentWaits[i] < recentWaits[j] })
		s.RecentPieceWaitP95Ms = float64(recentWaits[(95*len(recentWaits)+99)/100-1]) / float64(time.Millisecond)
	}
	if len(waits) == 0 {
		return s
	}
	sort.Slice(waits, func(i, j int) bool { return waits[i] < waits[j] })
	percentile := func(p int) float64 {
		index := (p*len(waits)+99)/100 - 1
		if index < 0 {
			index = 0
		}
		return float64(waits[index]) / float64(time.Millisecond)
	}
	s.PieceWaitP50Ms, s.PieceWaitP95Ms, s.PieceWaitP99Ms = percentile(50), percentile(95), percentile(99)
	return s
}
