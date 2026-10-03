package flow

import (
	"testing"
	"time"
)

func TestRecentCountersExpireWithoutLosingTotals(t *testing.T) {
	var c Counters
	c.Hit(128)
	c.Miss(64)
	c.Wait(time.Second)
	first := c.Snapshot()
	if first.RecentCacheHitBytes != 128 || first.RecentServerReadStalls != 1 {
		t.Fatal(first)
	}
	later := c.snapshotAt(time.Now().Add(61 * time.Second))
	if later.RecentCacheHitBytes != 0 || later.RecentPieceWaitP95Ms != 0 || later.PieceWaitCount != 1 || later.CacheHitBytes != 128 {
		t.Fatal(later)
	}
}

func TestWindowHysteresisBoundsAndSeekReset(t *testing.T) {
	now := time.Now()
	var h WindowSmoother
	if h.Apply(40, 100, now) != 40 {
		t.Fatal("initial window")
	}
	if h.Apply(90, 100, now.Add(time.Second)) != 50 {
		t.Fatal("growth not bounded")
	}
	if h.Apply(30, 100, now.Add(2*time.Second)) != 50 {
		t.Fatal("shrink should be held")
	}
	if h.Apply(30, 100, now.Add(11*time.Second)) != 38 {
		t.Fatal("shrink not bounded")
	}
	if h.Apply(90, 12, now.Add(12*time.Second)) > 12 {
		t.Fatal("budget exceeds cap")
	}
	h.Reset()
	if h.Apply(8, 100, now) != 8 {
		t.Fatal("seek/new file reset")
	}
}
