package flow

import (
	"testing"
	"time"
)

func TestCounterQuantilesAndBounds(t *testing.T) {
	var c Counters
	c.Hit(10)
	c.Miss(3)
	for i := 1; i <= 100; i++ {
		c.Wait(time.Duration(i) * time.Millisecond)
	}
	s := c.Snapshot()
	if s.CacheHitBytes != 10 || s.CacheMissBytes != 3 || s.PieceWaitCount != 100 || s.PieceWaitP50Ms != 50 || s.PieceWaitP95Ms != 95 || s.PieceWaitP99Ms != 99 {
		t.Fatalf("snapshot: %+v", s)
	}
	for i := 0; i < 1000; i++ {
		c.Wait(time.Millisecond)
	}
	if c.waitsUsed != len(c.waits) {
		t.Fatalf("ring grew: %d", c.waitsUsed)
	}
}
