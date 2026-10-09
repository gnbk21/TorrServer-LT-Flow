package flow

import (
	"math"
	"testing"
)

func TestDemandDeadlineUsesBytesAndQualifiedRate(t *testing.T) {
	if due, ok := DemandDeadline(0, 16*MiB, 1); !ok || due != 0 {
		t.Fatal("blocked piece is not immediate")
	}
	small, _ := DemandDeadline(1, MiB, float64(MiB))
	large, _ := DemandDeadline(1, 8*MiB, float64(MiB))
	fast, _ := DemandDeadline(1, 8*MiB, float64(4*MiB))
	if small != 50 || large != 7000 || fast != 1000 {
		t.Fatalf("deadlines: %d %d %d", small, large, fast)
	}
	if far, _ := DemandDeadline(1000000, 16*MiB, 1); far != 30000 {
		t.Fatal("deadline horizon unbounded")
	}
	for _, invalid := range []float64{0, -1, math.NaN(), math.Inf(1)} {
		if _, ok := DemandDeadline(1, MiB, invalid); ok {
			t.Fatal("unqualified rate accepted")
		}
	}
}

func TestScarceHintNeverOvertakesBlockedDemand(t *testing.T) {
	if ScarceDeadline(0, 0, true, true, true) != 0 || ScarceDeadline(1, 1200, true, true, true) != 200 || ScarceDeadline(2, 500, true, true, true) != 50 {
		t.Fatal("scarce lead bounds")
	}
	for _, input := range []struct {
		position              int
		sole, variable, fresh bool
	}{{5, true, true, true}, {1, false, true, true}, {1, true, false, true}, {1, true, true, false}} {
		if ScarceDeadline(input.position, 2500, input.sole, input.variable, input.fresh) != 2500 {
			t.Fatal("unqualified scarce evidence changed demand")
		}
	}
}
