package flow

import (
	"testing"
	"time"
)

func TestSlowUsefulStartupGetsBoundedProgressRenewal(t *testing.T) {
	start := time.Unix(100, 0)
	w := NewStartupWait(start, true)
	for second := 0; second < 300; second++ {
		if w.Expired(start.Add(time.Duration(second)*time.Second), int64(second)*256<<10, second >= 64) {
			t.Fatalf("steady verified progress expired at %d", second)
		}
	}
	if !w.Expired(start.Add(300*time.Second), 300*256<<10, true) || w.Budget() != StartupWaitMaximum {
		t.Fatal("startup exceeded its hard maximum")
	}
	if len(w.points) > 8 {
		t.Fatal("observation history grew unbounded")
	}
}

func TestStartupWaitDoesNotRewardStallsOrUnverifiedData(t *testing.T) {
	start := time.Unix(100, 0)
	for _, tc := range []struct {
		name               string
		adaptive, verified bool
		stopAt             int
	}{
		{"ordinary profile", false, true, 300},
		{"unverified opening", true, false, 300},
		{"stalled", true, true, 80},
		{"trickle", true, true, 300},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := NewStartupWait(start, tc.adaptive)
			for second := 0; second <= 120; second++ {
				bytes := int64(min(second, tc.stopAt)) * 256 << 10
				if tc.name == "trickle" {
					bytes = int64(second) * 1024
				}
				expired := w.Expired(start.Add(time.Duration(second)*time.Second), bytes, tc.verified)
				if expired != (second == 120) {
					t.Fatalf("unexpected expiry at %d: %v", second, expired)
				}
			}
			if w.Budget() != StartupWaitBase {
				t.Fatal("unearned extension")
			}
		})
	}
}

func TestStartupRenewalEndsWhenProgressStops(t *testing.T) {
	start := time.Unix(100, 0)
	w := NewStartupWait(start, true)
	for second := 0; second <= 150; second++ {
		expired := w.Expired(start.Add(time.Duration(second)*time.Second), int64(min(second, 121))*256<<10, true)
		if expired != (second == 150) {
			t.Fatalf("expiry at %d: %v", second, expired)
		}
	}
}
