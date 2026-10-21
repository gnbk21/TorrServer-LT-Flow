package flow

import (
	"testing"
	"time"
)

func TestSourceHealthBoundsAndPolicyIsolation(t *testing.T) {
	var h SourceHealth
	now := time.Unix(1000, 0)
	h.Observe("service-policy-A", true, now)
	if h.Allowed("service-policy-A", now) || !h.Allowed("service-policy-B", now) {
		t.Fatal("cooldown leaked across policies")
	}
	if !h.Allowed("service-policy-A", now.Add(16*time.Second)) {
		t.Fatal("circuit never recovered")
	}
	h.Observe("service-policy-A", false, now)
	if !h.Allowed("service-policy-A", now) {
		t.Fatal("success did not reset")
	}
	for i := 0; i < 100; i++ {
		h.Observe(string(rune(i)), true, now.Add(time.Duration(i)*time.Second))
	}
	if len(h.entries) != 64 {
		t.Fatal("health map unbounded")
	}
}
