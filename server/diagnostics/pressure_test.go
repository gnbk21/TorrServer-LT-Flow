package diagnostics

import (
	"testing"
	"testing/synctest"
	"time"
)

func TestPressureRecoveryNeedsThreeFreshSamples(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var p pressurePolicy
		total := uint64(16 << 30)
		threshold := pressureEntry(total)
		p.sample(total, threshold-1, true, time.Now())
		if !p.active {
			t.Fatal("entry threshold changed")
		}
		for i := 0; i < 50; i++ {
			p.sample(total, 2*threshold, true, time.Now())
		}
		if p.recovering != 0 {
			t.Fatal("cached sample advanced recovery")
		}
		for i := 1; i <= 2; i++ {
			time.Sleep(5 * time.Second)
			p.sample(total, 2*threshold, true, time.Now())
			if !p.active || p.recovering != i {
				t.Fatal("early recovery")
			}
		}
		time.Sleep(5 * time.Second)
		p.sample(total, threshold+threshold/2, true, time.Now())
		if p.recovering != 0 || !p.active {
			t.Fatal("threshold oscillation counted as recovery")
		}
		for i := 0; i < 3; i++ {
			time.Sleep(5 * time.Second)
			p.sample(total, 2*threshold, true, time.Now())
		}
		if p.active || p.state(true) != "normal" {
			t.Fatal("never recovered")
		}
		time.Sleep(5 * time.Second)
		p.sample(total, threshold-1, true, time.Now())
		time.Sleep(5 * time.Second)
		p.sample(total, 2*threshold, true, time.Now())
		time.Sleep(5 * time.Second)
		p.sample(0, 0, false, time.Now())
		if !p.active || p.recovering != 0 {
			t.Fatal("missing evidence cleared pressure")
		}
	})
}
