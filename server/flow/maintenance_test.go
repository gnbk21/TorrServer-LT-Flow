package flow

import (
	"testing"
	"time"
)

func TestMaintenanceNeverInterruptsAcceptedWork(t *testing.T) {
	var g RequestGate
	if !g.Enter() {
		t.Fatal("request denied")
	}
	if token, n := g.Acquire(time.Minute); token != "" || n != 1 {
		t.Fatal("quiesced an active stream")
	}
	g.Leave()
	token, n := g.Acquire(time.Minute)
	if token == "" || n != 0 {
		t.Fatal("idle gate failed")
	}
	if g.Enter() {
		t.Fatal("accepted playback during update")
	}
	if other, _ := g.Acquire(0); other != "" {
		t.Fatal("another owner stole the lease")
	}
	if g.Release("wrong") {
		t.Fatal("wrong owner released the lease")
	}
	g.Release(token)
	if !g.Enter() {
		t.Fatal("rollback did not reopen gate")
	}
	g.Leave()
}

func TestMaintenanceLeaseExpires(t *testing.T) {
	var g RequestGate
	token, _ := g.Acquire(time.Minute)
	g.expires = time.Now().Add(-time.Second)
	if !g.Enter() {
		t.Fatal("expired updater lease blocked playback")
	}
	if g.Release(token) {
		t.Fatal("expired lease still owns gate")
	}
	g.Leave()
}
