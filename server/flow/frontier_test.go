package flow

import (
	"math"
	"testing"
	"time"
)

func TestContiguousFrontierAndDiscontinuities(t *testing.T) {
	var f FrontierGrowth
	now := time.Unix(100, 0)
	if f.Observe(0, 100, true, now) != nil {
		t.Fatal("invented initial rate")
	}
	r := f.Observe(50, 150, true, now.Add(time.Second))
	if r == nil || *r != 50 {
		t.Fatal("counted HTTP progress instead of frontier growth", r)
	}
	if f.Observe(1000, 1100, true, now.Add(2*time.Second)) != nil {
		t.Fatal("seek counted as growth")
	}
	if f.Observe(1000, 1200, false, now.Add(3*time.Second)) != nil {
		t.Fatal("full/idle counted as supply")
	}
	if f.Observe(1000, 1300, true, now.Add(4*time.Second)) != nil {
		t.Fatal("idle history retained")
	}
	if f.Observe(1000, 1400, true, now.Add(12*time.Second)) != nil {
		t.Fatal("stale history retained")
	}
}

func TestUrgentHorizonScalesAndFallsBack(t *testing.T) {
	if got := UrgentHorizon(15e6, 4*MiB, 100, 1000); got != 10 {
		t.Fatal(got)
	}
	if got := UrgentHorizon(15e6, MiB, 100, 8000); got != 100 {
		t.Fatal("high latency starved pipeline", got)
	}
	if got := UrgentHorizon(1, 128*MiB, 100, 100); got < 2 {
		t.Fatal("one-piece bottleneck")
	}
	for _, rate := range []float64{0, -1, math.NaN(), math.Inf(1)} {
		if got := UrgentHorizon(rate, MiB, 100, 100); got != 100 {
			t.Fatal("invalid demand changed baseline", got)
		}
	}
	if got := UrgentHorizon(1e6, MiB, 1, 100); got != 1 {
		t.Fatal("exceeded tiny cache")
	}
}

func TestFrontierFrequentSamplesAccumulate(t *testing.T) {
	var f FrontierGrowth
	now := time.Unix(100, 0)
	f.Observe(0, 100, true, now)
	for i := 1; i < 5; i++ {
		if f.Observe(0, int64(100+i*10), true, now.Add(time.Duration(i)*100*time.Millisecond)) != nil {
			t.Fatal("sample too short")
		}
	}
	rate := f.Observe(0, 150, true, now.Add(500*time.Millisecond))
	if rate == nil || *rate != 100 {
		t.Fatal("frequent samples erased growth", rate)
	}
}
