package flow

import (
	"testing"
	"time"
)

func TestIntegrityWindowsDoNotCarryAcrossTorrentLifetimes(t *testing.T) {
	var meter IntegrityMeter
	now := time.Unix(1000, 0)
	meter.Record(now)
	meter.Record(now.Add(time.Second))
	if s := meter.Snapshot(now.Add(time.Second)); s.Failures != 2 || s.RecentFailures != 2 {
		t.Fatal(s)
	}
	if s := meter.Snapshot(now.Add(61 * time.Second)); s.Failures != 2 || s.RecentFailures != 0 {
		t.Fatal(s)
	}
	var next IntegrityMeter
	if s := next.Snapshot(now); s.Failures != 0 || s.RecentFailures != 0 || s.LastFailureAtMs != 0 {
		t.Fatal(s)
	}
}
