package flow

import (
	"math"
	"testing"
	"time"
)

func TestStartupReserveWaitRejectsExpensiveWholePieceExtension(t *testing.T) {
	supply := reserveSupply()
	supply.LongRate = 762600.7272727273
	plan := PlanStartupReserve(32*MiB, 64*MiB, 16*MiB, 2983181.375, 45, supply)
	if plan.TargetBytes != 64*MiB || *plan.AdditionalFillSeconds < 40 {
		t.Fatal("recorded slow-start condition was not reproduced", plan)
	}
	var wait StartupReserveWait
	limited := wait.Limit(plan, 8*MiB, 16*MiB, false, time.Unix(100, 0))
	if limited.TargetBytes != 32*MiB || !limited.ExtraWaitLimited || limited.RequestedBytes != 64*MiB ||
		limited.ExtraWaitBudgetMs != 10000 || limited.AdditionalFillSeconds == nil || *limited.AdditionalFillSeconds != 0 {
		t.Fatal("optional reserve still adds a slow whole-piece download", limited)
	}
}

func TestStartupReserveWaitCountsAlignmentAndRetainsAffordableInsurance(t *testing.T) {
	for _, tc := range []struct {
		base, target, offset, piece int64
		rate                        float64
		want                        int64
	}{
		{32 * MiB, 32*MiB + 1, 0, 16 * MiB, float64(MiB), 32 * MiB},
		{32 * MiB, 40 * MiB, 8 * MiB, 16 * MiB, float64(MiB), 40 * MiB},
		{32 * MiB, 64 * MiB, 8 * MiB, 16 * MiB, float64(2 * MiB), 48 * MiB},
		{32 * MiB, 64 * MiB, 0, 16 * MiB, float64(8 * MiB), 64 * MiB},
		{96 * MiB, 128 * MiB, 0, 16 * MiB, float64(MiB), 96 * MiB},
	} {
		var wait StartupReserveWait
		p := wait.Limit(ReservePlan{Known: true, BaseBytes: tc.base, TargetBytes: tc.target,
			MaximumBytes: tc.target, DeliveryRate: tc.rate}, tc.offset, tc.piece, false, time.Unix(100, 0))
		if p.TargetBytes != tc.want || p.TargetBytes < tc.base || p.TargetBytes > tc.target ||
			p.AdditionalFillSeconds == nil || *p.AdditionalFillSeconds > 10 {
			t.Fatalf("whole-piece budget: %+v -> %+v", tc, p)
		}
	}
}

func TestStartupReserveWaitExpiresOnlyAfterRequiredHeadAndIndexAreReady(t *testing.T) {
	var wait StartupReserveWait
	now := time.Unix(100, 0)
	plan := ReservePlan{Known: true, BaseBytes: 96 * MiB, TargetBytes: 128 * MiB,
		MaximumBytes: 128 * MiB, DeliveryRate: float64(8 * MiB)}
	if p := wait.Limit(plan, 0, 16*MiB, false, now); p.TargetBytes != plan.TargetBytes {
		t.Fatal(p)
	}
	if p := wait.Limit(plan, 0, 16*MiB, false, now.Add(time.Minute)); p.ExtraWaitExpired {
		t.Fatal("missing required data was treated as a ready base", p)
	}
	ready := now.Add(time.Minute)
	wait.Limit(plan, 0, 16*MiB, true, ready)
	if p := wait.Limit(plan, 0, 16*MiB, true, ready.Add(10*time.Second-time.Millisecond)); p.TargetBytes != plan.TargetBytes {
		t.Fatal("affordable reserve was cut short", p)
	}
	for _, at := range []time.Time{ready.Add(10 * time.Second), ready.Add(time.Minute)} {
		p := wait.Limit(plan, 0, 16*MiB, true, at)
		if !p.ExtraWaitExpired || !p.ExtraWaitLimited || p.TargetBytes != 96*MiB {
			t.Fatal("optional wait did not end at the high-bitrate base", p)
		}
	}
}

func TestStartupReserveWaitHandlesUnknownEvidenceAndClockChanges(t *testing.T) {
	plan := ReservePlan{Known: true, BaseBytes: 32 * MiB, TargetBytes: 64 * MiB,
		MaximumBytes: 64 * MiB, DeliveryRate: float64(8 * MiB)}
	for _, rate := range []float64{0, -1, math.NaN(), math.Inf(1)} {
		var wait StartupReserveWait
		plan.DeliveryRate = rate
		if p := wait.Limit(plan, 0, 16*MiB, true, time.Unix(100, 0)); p.TargetBytes != plan.BaseBytes {
			t.Fatal("unknown delivery admitted more startup work", p)
		}
	}
	plan.DeliveryRate = float64(8 * MiB)
	var wait StartupReserveWait
	wait.Limit(plan, 0, 16*MiB, true, time.Unix(100, 0))
	if p := wait.Limit(plan, 0, 16*MiB, true, time.Unix(90, 0)); p.ExtraWaitExpired {
		t.Fatal("backwards clock expired the reserve", p)
	}
	wait.Limit(plan, 0, 16*MiB, false, time.Unix(95, 0))
	if p := wait.Limit(plan, 0, 16*MiB, true, time.Unix(110, 0)); p.ExtraWaitExpired {
		t.Fatal("lost base residency did not restart the ready observation", p)
	}
}
