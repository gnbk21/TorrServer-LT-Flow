package flow

import "testing"

func TestStartupTarget(t *testing.T) {
	if got := StartupTarget(Provisional(), 6, 32, 128, 130); got != 32*MiB {
		t.Fatalf("provisional = %d", got)
	}
	if got := StartupTarget(Estimate{BytesPerSecond: 100e6 / 8}, 6, 32, 128, 130); got != 97500000 {
		t.Fatalf("100 Mbps = %d", got)
	}
	if got := StartupTarget(Estimate{BytesPerSecond: 1e9}, 6, 32, 128, 130); got != 128*MiB {
		t.Fatalf("max = %d", got)
	}
}

func TestEstimateAndBuffer(t *testing.T) {
	e := MediaEstimate(7500000000, 600, "")
	if e.Source != "DERIVED" || e.BytesPerSecond != 12500000 {
		t.Fatalf("derived: %+v", e)
	}
	e = MediaEstimate(7500000000, 600, "100000000")
	if e.Source != "PROBED" || e.BytesPerSecond != 12500000 {
		t.Fatalf("probed: %+v", e)
	}
	if got := BufferSeconds(125000000, e); got != 10 {
		t.Fatalf("buffer: %v", got)
	}
}
