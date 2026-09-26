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
	waitsUsed   int
	waitsNext   int
}

type CounterSnapshot struct {
	CacheHitBytes       int64   `json:"cache_hit_bytes"`
	CacheMissBytes      int64   `json:"cache_miss_bytes"`
	PieceWaitCount      int64   `json:"piece_wait_count"`
	PieceWaitDurationMs float64 `json:"piece_wait_duration_ms"`
	PieceWaitP50Ms      float64 `json:"piece_wait_p50_ms"`
	PieceWaitP95Ms      float64 `json:"piece_wait_p95_ms"`
	PieceWaitP99Ms      float64 `json:"piece_wait_p99_ms"`
}

func (c *Counters) Hit(n int) {
	if n > 0 {
		c.hitBytes.Add(int64(n))
	}
}
func (c *Counters) Miss(n int) {
	if n > 0 {
		c.missBytes.Add(int64(n))
	}
}
func (c *Counters) Wait(d time.Duration) {
	if d < 0 {
		return
	}
	c.waitCount.Add(1)
	c.waitTotalNs.Add(int64(d))
	c.waitsMu.Lock()
	c.waits[c.waitsNext] = int64(d)
	c.waitsNext = (c.waitsNext + 1) % len(c.waits)
	if c.waitsUsed < len(c.waits) {
		c.waitsUsed++
	}
	c.waitsMu.Unlock()
}

func (c *Counters) Snapshot() CounterSnapshot {
	s := CounterSnapshot{CacheHitBytes: c.hitBytes.Load(), CacheMissBytes: c.missBytes.Load(), PieceWaitCount: c.waitCount.Load()}
	if s.PieceWaitCount > 0 {
		s.PieceWaitDurationMs = float64(c.waitTotalNs.Load()) / float64(time.Millisecond) / float64(s.PieceWaitCount)
	}
	c.waitsMu.Lock()
	waits := append([]int64(nil), c.waits[:c.waitsUsed]...)
	c.waitsMu.Unlock()
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
