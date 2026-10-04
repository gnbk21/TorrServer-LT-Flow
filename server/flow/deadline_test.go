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
