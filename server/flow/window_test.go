package flow

import (
	"testing"
	"time"
)

func TestAdaptiveWindowBoundsAndSwarm(t *testing.T) {
	const piece = 4 * MiB
	rate := float64(2 * MiB)
	_, healthy := AdaptiveWindow(rate, 5*rate, 0, 45, 180, 130, piece, 100)
	_, marginal := AdaptiveWindow(rate, 1.2*rate, 0, 45, 180, 130, piece, 100)
	_, weak := AdaptiveWindow(rate, 0.7*rate, 0, 45, 180, 130, piece, 100)
	if !(healthy < marginal && marginal < weak) {
		t.Fatalf("window did not grow as the swarm weakened: %d %d %d", healthy, marginal, weak)
	}
	if _, n := AdaptiveWindow(rate, 0, 0, 45, 180, 130, piece, 10); n != 10 {
		t.Fatalf("cache budget clamp = %d", n)
	}
	if _, n := AdaptiveWindow(0, rate, 0, 45, 180, 130, piece, 100); n != 0 {
		t.Fatalf("unknown bitrate should leave upstream window alone, got %d", n)
	}
}

func TestConsumptionTrackerIgnoresSeekAndIdleGap(t *testing.T) {
	var c ConsumptionTracker
	now := time.Unix(1000, 0)
	baseline := float64(MiB)
	c.Observe(0, now, baseline)
	c.Observe(2*MiB, now.Add(2*time.Second), baseline)
	c.Observe(4*MiB, now.Add(4*time.Second), baseline)
	c.Observe(6*MiB, now.Add(6*time.Second), baseline)
	if rate, confidence := c.Rate(); confidence != "stable" || rate < baseline/2 || rate > 2*baseline {
		t.Fatalf("rate=%v confidence=%s", rate, confidence)
	}
	c.Observe(100*MiB, now.Add(7*time.Second), baseline)  // seek
	c.Observe(101*MiB, now.Add(50*time.Second), baseline) // idle
	if rate, _ := c.Rate(); rate > 2*baseline {
		t.Fatalf("seek/idle poisoned observed rate: %v", rate)
	}
}
