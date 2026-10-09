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

func TestDeliveryStatsAndOutagesRespectIdleAndBudgets(t *testing.T) {
	var rates RateWindow
	now := time.Unix(1000, 0)
	for i := 0; i < 10; i++ {
		rate := float64(MiB)
		if i%2 == 0 {
			rate *= 8
		}
		rates.Observe(rate, now.Add(time.Duration(i)*time.Second))
	}
	stats := rates.Stats(now.Add(9 * time.Second))
	if stats.Samples != 10 || stats.Variation <= .5 {
		t.Fatalf("variation missing: %+v", stats)
	}
	for i := 10; i < 13; i++ {
		rates.Observe(0, now.Add(time.Duration(i)*time.Second))
	}
	stats = rates.Stats(now.Add(12 * time.Second))
	if stats.OutageSeconds != 3 {
		t.Fatalf("outage=%d", stats.OutageSeconds)
	}
	delivery := DeliveryStats{Samples: 10, OutageSeconds: 12}
	idle, _ := DeliveryWindow(float64(MiB), 0, delivery, false, true, 45, 180, 130, MiB, 200)
	blocked, pieces := DeliveryWindow(float64(MiB), 0, delivery, true, false, 45, 180, 130, MiB, 20)
	if idle != 45 || blocked != 102 || pieces > 20 {
		t.Fatalf("idle=%d blocked=%d pieces=%d", idle, blocked, pieces)
	}
	if seconds, _ := DeliveryWindow(float64(MiB), 0, delivery, true, false, 45, 60, 130, MiB, 200); seconds > 60 {
		t.Fatal("delivery adaptation exceeded time budget")
	}
	if stats := rates.Stats(now.Add(14 * time.Second)); stats.OutageSeconds != 0 {
		t.Fatal("unobserved seconds invented an outage")
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

func TestBufferExhaustionSeconds(t *testing.T) {
	if seconds, ok := BufferExhaustionSeconds(30*MiB, float64(2*MiB), float64(MiB)); !ok || seconds != 30 {
		t.Fatalf("draining buffer: seconds=%v known=%v", seconds, ok)
	}
	for _, rates := range [][2]float64{{0, 0}, {float64(MiB), float64(MiB)}, {float64(MiB), float64(2 * MiB)}} {
		if _, ok := BufferExhaustionSeconds(MiB, rates[0], rates[1]); ok {
			t.Fatalf("non-draining or unknown rates %v reported exhaustion", rates)
		}
	}
}
