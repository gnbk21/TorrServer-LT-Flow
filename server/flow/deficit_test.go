package flow

import (
	"math"
	"testing"
	"time"
)

func TestCumulativeDeficitWithContinuousSupply(t *testing.T) {
	var m DeliveryMeter
	start := time.Unix(1000, 0)
	for i := 0; i < 10; i++ {
		now := start.Add(time.Duration(i) * time.Second)
		m.ObserveDemand(now, "DEMAND", 11_250_000)
		m.AddVerified(7_500_000, now)
	}
	e := m.Snapshot(start.Add(10 * time.Second))
	if e.DeficitBytes != 37_500_000 || e.DeficitSamples != 10 || e.OutageSeconds != 0 {
		t.Fatal(e)
	}
	if got := DeficitTarget(e, 11_250_000, 45, 180); got != 50 {
		t.Fatal(got)
	}
	if got := DeficitStartupTarget(32*MiB, 128*MiB, e); got != 32*MiB+48_750_000 {
		t.Fatal(got)
	}
	if DeficitTarget(e, 11_250_000, 178, 180) != 180 || DeficitStartupTarget(32*MiB, 40*MiB, e) != 40*MiB {
		t.Fatal("reserve exceeded limit")
	}
	m.Reset()
	if m.Snapshot(start.Add(11*time.Second)).DeficitBytes != 0 {
		t.Fatal("seek retained deficit")
	}
}

func TestDeficitDoesNotBridgeUnqualifiedIntervals(t *testing.T) {
	for _, mode := range []string{"FULL", "IDLE", "PROBE", "PREPARATION", "RECONNECT"} {
		var m DeliveryMeter
		start := time.Unix(1000, 0)
		for i := 0; i < 9; i++ {
			now := start.Add(time.Duration(i) * time.Second)
			m.ObserveDemand(now, "DEMAND", 1000)
			m.AddVerified(500, now)
			if i == 4 {
				m.Observe(now, mode)
			}
		}
		e := m.Snapshot(start.Add(9 * time.Second))
		if e.DeficitBytes != 2000 {
			t.Fatalf("%s bridged gap: %+v", mode, e)
		}
		if m.Snapshot(start.Add(90*time.Second)).DeficitBytes != 0 {
			t.Fatal("deficit did not expire")
		}
	}
	for _, rate := range []float64{0, -1, math.NaN(), math.Inf(1)} {
		var m DeliveryMeter
		for i := 0; i < 10; i++ {
			m.ObserveDemand(time.Unix(1000+int64(i), 0), "DEMAND", rate)
		}
		if e := m.Snapshot(time.Unix(1010, 0)); e.DeficitBytes != 0 || e.DeficitSamples != 0 {
			t.Fatal(e)
		}
	}
}

func TestDeficitAccountsForSurplusAndChangingDemand(t *testing.T) {
	var m DeliveryMeter
	for i, delivery := range []int64{500, 500, 3000, 1500, 1500} {
		rate := float64(1000)
		if i >= 3 {
			rate = 2000
		}
		now := time.Unix(1000+int64(i), 0)
		m.ObserveDemand(now, "DEMAND", rate)
		m.AddVerified(delivery, now)
	}
	e := m.Snapshot(time.Unix(1005, 0))
	if e.DeficitBytes != 1000 {
		t.Fatal(e)
	}
	e.AgeMs = 3000
	if DeficitTarget(e, 1000, 45, 180) != 45 {
		t.Fatal("stale reserve enlarged target")
	}
}

func TestDeficitRequiresConsecutiveQualifiedIntervals(t *testing.T) {
	var m DeliveryMeter
	for i := 0; i < 9; i++ {
		mode := "DEMAND"
		if i%3 == 2 {
			mode = "PROBE"
		}
		m.ObserveDemand(time.Unix(1000+int64(i), 0), mode, 1000)
	}
	evidence := m.Snapshot(time.Unix(1009, 0))
	if evidence.DeficitSamples != 2 || evidence.DeficitBytes != 0 {
		t.Fatal("scattered intervals qualified a deficit", evidence)
	}
}
