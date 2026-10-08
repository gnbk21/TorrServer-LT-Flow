package flow

import (
	"math"
	"testing"
	"time"
)

func TestResilientDemandHonorsQualifiedAverageAndBounds(t *testing.T) {
	e := Estimate{BytesPerSecond: 12e6, Confidence: "medium"}
	for _, tc := range []struct {
		observed   float64
		confidence string
		want       float64
	}{
		{8e6, "stable", 12e6}, {18e6, "stable", 18e6}, {100e6, "stable", 24e6},
		{18e6, "low", 12e6}, {math.NaN(), "stable", 12e6}, {math.Inf(1), "stable", 12e6},
	} {
		if got := ResilientDemand(e, tc.observed, tc.confidence); got != tc.want {
			t.Fatalf("%+v: %f", tc, got)
		}
	}
	if ResilientDemand(Provisional(), 100e6, "stable") != 0 {
		t.Fatal("invented media demand")
	}
}

func TestRecentOutageReservePersistsAndExpires(t *testing.T) {
	var meter DeliveryMeter
	start := time.Unix(5000, 0)
	for i := 0; i < 15; i++ {
		now := start.Add(time.Duration(i) * time.Second)
		meter.Observe(now, "DEMAND")
		if i < 5 || i >= 9 {
			meter.AddVerified(12e6, now)
		}
	}
	e := meter.Snapshot(start.Add(15 * time.Second))
	if e.OutageSeconds != 0 || e.RecentOutageSeconds != 4 || ResilienceTarget(e, 45, 180) != 53 {
		t.Fatal(e)
	}
	if ResilienceTarget(e, 175, 180) != 180 {
		t.Fatal("maximum exceeded")
	}
	for _, mode := range []string{"FULL", "IDLE", "PROBE", "RECONNECT"} {
		e.Mode = mode
		if ResilienceTarget(e, 45, 180) != 45 {
			t.Fatal("unqualified reserve", mode)
		}
	}
	meter.Reset()
	if meter.Snapshot(start.Add(16*time.Second)).RecentOutageSeconds != 0 {
		t.Fatal("seek retained outage")
	}
	if ResilienceTarget(DeliveryEvidence{Mode: "DEMAND", Confidence: "high", AgeMs: 3000, RecentOutageSeconds: 20}, 45, 180) != 45 {
		t.Fatal("stale reserve")
	}
}

func TestObservedRateExpiresAfterIdle(t *testing.T) {
	var tracker ConsumptionTracker
	start := time.Unix(1000, 0)
	for i := 0; i < 5; i++ {
		tracker.Observe(int64(i)*8*MiB, start.Add(time.Duration(i)*2*time.Second), float64(4*MiB))
	}
	if _, confidence := tracker.RateAt(start.Add(9 * time.Second)); confidence != "stable" {
		t.Fatal("rate did not qualify")
	}
	if rate, _ := tracker.RateAt(start.Add(40 * time.Second)); rate != 0 {
		t.Fatal("idle retained live demand")
	}
	tracker.Observe(33*MiB, start.Add(40*time.Second), float64(4*MiB))
	if rate, confidence := tracker.RateAt(start.Add(40 * time.Second)); rate != 0 || confidence == "stable" {
		t.Fatal("first read after idle resurrected stale rate")
	}
}

func TestUnknownMediaHighBitrateSequentialProgressQualifies(t *testing.T) {
	for _, bytesPerSecond := range []int64{11_250_000, 15_000_000, 25_000_000} { // 90, 120, 200 Mbps
		var tracker ConsumptionTracker
		start := time.Unix(1000, 0)
		for i := int64(0); i <= 1000; i++ {
			tracker.Observe(i*bytesPerSecond/100, start.Add(time.Duration(i)*10*time.Millisecond), 0)
		}
		rate, confidence := tracker.RateAt(start.Add(10 * time.Second))
		if confidence != "stable" || math.Abs(rate-float64(bytesPerSecond)) > float64(bytesPerSecond)*.01 {
			t.Fatalf("%d B/s sequential reads did not qualify: %f %s", bytesPerSecond, rate, confidence)
		}
		tracker.Observe(1<<30, start.Add(11*time.Second), 0)
		if rate, confidence := tracker.Rate(); rate != 0 || confidence == "stable" {
			t.Fatal("seek retained pre-seek confidence")
		}
	}
}
