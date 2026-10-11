package torrstor

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestEvictionWorkerRechecksChangesAfterNoProgress(t *testing.T) {
	c := &Cache{}
	entered, release, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var passes atomic.Int32
	pass := func() {
		if passes.Add(1) == 1 {
			close(entered)
			<-release // Old budget/protection snapshot cannot free any bytes.
		}
	}
	go func() { c.runEviction(pass); close(done) }()
	<-entered
	c.runEviction(pass) // Capacity/protection changed while the first pass ran.
	close(release)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("eviction worker did not drain")
	}
	if passes.Load() != 2 || c.evicting.Load() || c.evictAgain.Load() {
		t.Fatalf("queued change was not drained exactly once: passes=%d", passes.Load())
	}
}

func TestEvictionWorkerCoalescesConcurrentRequests(t *testing.T) {
	c := &Cache{}
	entered, release, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var passes, active, peak atomic.Int32
	pass := func() {
		current := active.Add(1)
		for previous := peak.Load(); current > previous; previous = peak.Load() {
			if peak.CompareAndSwap(previous, current) {
				break
			}
		}
		defer active.Add(-1)
		if passes.Add(1) == 1 {
			close(entered)
			<-release
		}
	}
	go func() { c.runEviction(pass); close(done) }()
	<-entered
	var callers sync.WaitGroup
	for range 64 {
		callers.Go(func() { c.runEviction(pass) })
	}
	callers.Wait()
	close(release)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("coalesced eviction worker did not drain")
	}
	if passes.Load() != 2 || peak.Load() != 1 || c.evicting.Load() || c.evictAgain.Load() {
		t.Fatalf("requests were lost or ran concurrently: passes=%d peak=%d", passes.Load(), peak.Load())
	}
}
