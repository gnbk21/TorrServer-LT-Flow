package flow

import (
	"testing"
	"time"
)

func TestQualifiedDelivery(t *testing.T) {
	var m DeliveryMeter
	start := time.Unix(1000, 0)
	for i := 0; i < 12; i++ {
		now := start.Add(time.Duration(i) * time.Second)
		m.Observe(now, "DEMAND")
		m.AddVerified(1000, now)
	}
	e := m.Snapshot(start.Add(12 * time.Second))
	if e.LongRate != 1000 || e.Confidence != "high" {
		t.Fatalf("%+v", e)
	}
	m.Observe(start.Add(12*time.Second), "FULL")
	m.AddVerified(999999, start.Add(12*time.Second))
	e = m.Snapshot(start.Add(13 * time.Second))
	if e.Confidence != "unknown" || e.LongRate != 1000 {
		t.Fatalf("idle polluted supply: %+v", e)
	}
	m.Reset()
	if m.Snapshot(start.Add(14*time.Second)).Samples != 0 {
		t.Fatal("seek retained samples")
	}
}
func TestRiskRequiresQualifiedEvidence(t *testing.T) {
	unknown := BufferRisk(0, 1000, 900, nil, DeliveryEvidence{Confidence: "unknown"}, 45, 180)
	if unknown.Level != "UNKNOWN" || unknown.ExhaustionSeconds != nil {
		t.Fatal(unknown)
	}
	r := BufferRisk(5000, 1000, 900, nil, DeliveryEvidence{Confidence: "high", LongRate: 500, ShortRate: 500, ShortSamples: 5}, 45, 90)
	if r.Level != "HIGH" || r.TargetSeconds != 90 || r.ExhaustionSeconds == nil || *r.ExhaustionSeconds != 10 {
		t.Fatal(r)
	}
}

func TestDeliveryExcludesMixedAndStaleIntervals(t *testing.T) {
	var m DeliveryMeter
	now := time.Unix(2000, 0)
	m.Observe(now, "FULL")
	m.Observe(now.Add(time.Millisecond), "DEMAND")
	m.AddVerified(1000, now.Add(2*time.Millisecond))
	if e := m.Snapshot(now.Add(time.Second)); e.Samples != 0 {
		t.Fatal("mixed interval counted", e)
	}
	for _, mode := range []string{"IDLE", "PROBE", "PREPARATION", "RECONNECT"} {
		m.Observe(now, mode)
		m.AddVerified(1000, now)
		if m.Snapshot(now.Add(time.Second)).Samples != 0 {
			t.Fatal(mode)
		}
	}
	m.Reset()
	m.Observe(now, "DEMAND")
	m.AddVerified(1000, now.Add(3*time.Second))
	if e := m.Snapshot(now.Add(4 * time.Second)); e.LongRate != 0 || e.Confidence != "unknown" {
		t.Fatal("stale classifier counted", e)
	}
	m.Reset()
	for i := 0; i < 12; i++ {
		m.Observe(now.Add(time.Duration(i)*time.Second), "DEMAND")
	}
	if e := m.Snapshot(now.Add(12 * time.Second)); e.OutageSeconds != 12 || e.Confidence != "high" {
		t.Fatal("active supply outage lost", e)
	}
	if e := m.Snapshot(now.Add(2 * time.Minute)); e.Samples != 0 || e.Confidence != "unknown" {
		t.Fatal("expired supply retained", e)
	}
}
