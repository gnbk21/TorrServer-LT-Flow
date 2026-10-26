package flow

import (
	"encoding/json"
	"math"
	"testing"
)

func reserveSupply() DeliveryEvidence {
	return DeliveryEvidence{Mode: "DEMAND", Confidence: "high", Samples: 20, ShortSamples: 5,
		AgeMs: 100, LongRate: 2e6, ShortRate: 2e6, DeficitSamples: 20, DeficitBytes: 1e6}
}

func TestStartupReserveProjectsMeasuredDeficitAcrossBitrates(t *testing.T) {
	for _, demand := range []float64{125_000, 3e6, 12.5e6, 25e6} {
		e := reserveSupply()
		e.LongRate = demand * .8
		base, maximum := int64(32*MiB), int64(256*MiB)
		plan := PlanStartupReserve(base, maximum, 16<<10, demand, 45, e)
		want := min(maximum, base+int64(math.Ceil(max(1e6, demand*.2*45)*1.3)))
		if !plan.Known || plan.TargetBytes != want || plan.Reason != "SUPPLY_DEFICIT" {
			t.Fatalf("demand=%f: %+v, want %d", demand, plan, want)
		}
		if plan.AdditionalFillSeconds == nil || *plan.AdditionalFillSeconds <= 0 {
			t.Fatal("missing bounded fill estimate", plan)
		}
	}
}

func TestStartupReserveKeepsFastUnknownAndIdlePaths(t *testing.T) {
	for _, mutate := range []func(*DeliveryEvidence){
		func(e *DeliveryEvidence) { e.Mode = "FULL" },
		func(e *DeliveryEvidence) { e.Mode = "PROBE" },
		func(e *DeliveryEvidence) { e.Mode = "PREPARATION" },
		func(e *DeliveryEvidence) { e.Mode = "IDLE" },
		func(e *DeliveryEvidence) { e.Mode = "RECONNECT" },
		func(e *DeliveryEvidence) { e.AgeMs = 2501 },
		func(e *DeliveryEvidence) { e.AgeMs = -1 },
		func(e *DeliveryEvidence) { e.Confidence = "unknown" },
		func(e *DeliveryEvidence) { e.Samples = 2 },
		func(e *DeliveryEvidence) { e.LongRate = math.NaN() },
		func(e *DeliveryEvidence) { e.LongRate = math.Inf(1) },
	} {
		e := reserveSupply()
		mutate(&e)
		p := PlanStartupReserve(32*MiB, 128*MiB, 16<<10, 3e6, 45, e)
		if p.Known || p.TargetBytes != 32*MiB {
			t.Fatal("unqualified supply extended startup", p)
		}
	}
	e := reserveSupply()
	e.LongRate, e.DeficitBytes = 10e6, 0
	if p := PlanStartupReserve(32*MiB, 128*MiB, 16<<10, 3e6, 45, e); !p.Known || p.TargetBytes != 32*MiB {
		t.Fatal("healthy supply extended startup", p)
	}
	for _, demand := range []float64{0, -1, math.NaN(), math.Inf(1)} {
		if p := PlanStartupReserve(32*MiB, 128*MiB, 16<<10, demand, 45, e); p.Known || p.TargetBytes != 32*MiB {
			t.Fatal("unknown media demand extended startup", p)
		}
	}
}

func TestStartupReserveHonorsLimitsAndZeroSupply(t *testing.T) {
	e := reserveSupply()
	e.LongRate = 0
	p := PlanStartupReserve(32*MiB, 40*MiB, 16<<10, 12.5e6, 180, e)
	if !p.Limited || p.TargetBytes != 40*MiB || p.ForecastSeconds != 60 || p.AdditionalFillSeconds != nil {
		t.Fatal(p)
	}
	for _, maximum := range []int64{0, 32 * MiB, 64 * MiB, math.MaxInt64} {
		p = PlanStartupReserve(32*MiB, maximum, 16<<10, math.MaxFloat64, 600, e)
		if p.TargetBytes < p.BaseBytes || p.TargetBytes > max(maximum, p.BaseBytes) {
			t.Fatal("reserve overflow", p)
		}
		if _, err := json.Marshal(p); err != nil {
			t.Fatal("non-finite diagnostic", err)
		}
	}
}

func TestStartupReserveWaitsForPieceScaleSupplyEvidence(t *testing.T) {
	for _, tc := range []struct {
		piece   int64
		demand  float64
		samples int
	}{
		{256 << 10, 125000, 5}, {256 << 10, 3e6, 3}, {256 << 10, 12.5e6, 3},
		{4 << 20, 125000, 60}, {4 << 20, 3e6, 3}, {4 << 20, 12.5e6, 3},
		{16 << 20, 125000, 60}, {16 << 20, 3e6, 12}, {16 << 20, 12.5e6, 3},
	} {
		e := reserveSupply()
		e.Samples, e.LongRate = tc.samples-1, 0 // payload may still be arriving
		p := PlanStartupReserve(32*MiB, 128*MiB, tc.piece, tc.demand, 45, e)
		if p.Known || p.TargetBytes != p.BaseBytes {
			t.Fatalf("premature reserve for piece=%d demand=%f: %+v", tc.piece, tc.demand, p)
		}
		e.Samples, e.DeficitSamples = tc.samples, tc.samples
		if p = PlanStartupReserve(32*MiB, 128*MiB, tc.piece, tc.demand, 45, e); !p.Known || p.TargetBytes <= p.BaseBytes {
			t.Fatalf("mature deficit ignored for piece=%d demand=%f: %+v", tc.piece, tc.demand, p)
		}
	}
	if p := PlanStartupReserve(32*MiB, 128*MiB, 0, 3e6, 45, reserveSupply()); p.Known {
		t.Fatal("unknown piece size qualified startup supply")
	}
}

func TestBufferSupplySeparatesDeficitGapsAndUnknownPlayerBuffer(t *testing.T) {
	e := reserveSupply()
	s := AssessBufferSupply(3e6, 30e6, e)
	if !s.Known || s.DeficitRate != 1e6 || s.Runway == nil || *s.Runway != 30 {
		t.Fatal(s)
	}
	e.LongRate, e.ShortRate = 4e6, 0
	if s = AssessBufferSupply(3e6, 30e6, e); s.State != "RECENT_GAPS" || s.Runway != nil {
		t.Fatal(s)
	}
	e.AgeMs = 3000
	if s = AssessBufferSupply(3e6, 30e6, e); s.Known || s.State != "UNKNOWN" {
		t.Fatal(s)
	}
}
