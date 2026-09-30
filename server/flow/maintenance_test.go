package flow

import "testing"

func TestMaintenanceNeverInterruptsAcceptedWork(t *testing.T) {
	var g RequestGate
	if !g.Enter() {
		t.Fatal("request denied")
	}
	if enabled, n := g.Set(true); enabled || n != 1 {
		t.Fatal("quiesced an active stream")
	}
	g.Leave()
	if enabled, n := g.Set(true); !enabled || n != 0 {
		t.Fatal("idle gate failed")
	}
	if g.Enter() {
		t.Fatal("accepted playback during update")
	}
	g.Set(false)
	if !g.Enter() {
		t.Fatal("rollback did not reopen gate")
	}
	g.Leave()
}
