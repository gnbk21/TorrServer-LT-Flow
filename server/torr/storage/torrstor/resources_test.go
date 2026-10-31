package torrstor

import (
	"bytes"
	"encoding/json"
	"io"
	"server/diagnostics"
	"server/flow"
	"server/settings"
	"testing"
	"time"
)

func TestResourceThrottlePrecedesMemorySampling(t *testing.T) {
	resourceState.Lock()
	old := resourceState.status
	now := time.Now()
	resourceState.status = ResourceStatus{SampledAt: now}
	resourceState.Unlock()
	t.Cleanup(func() { resourceState.Lock(); resourceState.status = old; resourceState.Unlock() })
	s := NewStorage()
	calls := 0
	sample := func() diagnostics.MemoryStatus { calls++; return diagnostics.MemoryStatus{} }
	for i := 0; i < 1000; i++ {
		s.reconcileResourcesSample(sample, now.Add(999*time.Millisecond))
	}
	if calls != 0 {
		t.Fatal("throttled native updates still sampled memory", calls)
	}
	s.reconcileResourcesSample(sample, now.Add(time.Second))
	if calls != 1 {
		t.Fatal("due reconciliation did not sample memory", calls)
	}
}

func TestAllocationRAMExcludesDiskBackedCache(t *testing.T) {
	old := settings.BTsets()
	settings.StoreBTsets(settings.NewDefaultConfig())
	t.Cleanup(func() { settings.StoreBTsets(old) })
	s := NewStorage()
	s.callbackOpen(101, mkHash(101), 1, pieceLen)
	s.callbackOpen(102, mkHash(102), 1, pieceLen)
	t.Cleanup(func() { s.callbackClose(101); s.callbackClose(102) })
	ram, disk := s.lookup(101), s.lookup(102)
	disk.diskRoot = t.TempDir()
	for _, c := range []*Cache{ram, disk} {
		if n, err := c.writePiece(0, 0, bytes.Repeat([]byte{1}, int(pieceLen))); err != nil || int64(n) != pieceLen {
			t.Fatal(n, err)
		}
	}
	allocation := s.Allocations()
	if allocation.ResidentBytes != 2*pieceLen || allocation.RAMResidentBytes != pieceLen {
		t.Fatal("disk payload counted as RAM", allocation)
	}
}

func TestPressureTransitionsProtectDemandAndRecoverBackground(t *testing.T) {
	old := settings.BTsets()
	f := settings.DefaultFlowSettings()
	f.GlobalCacheBudgetMB = 256
	f.WarmCacheBudgetMB = 64
	settings.StoreBTsets(&settings.BTSets{CacheSize: 256 * flow.MiB, ReaderReadAHead: 70, Flow: f})
	reset := func() { resourceState.Lock(); resourceState.status = ResourceStatus{}; resourceState.Unlock() }
	reset()
	t.Cleanup(func() { settings.StoreBTsets(old); reset() })
	s := NewStorage()
	s.callbackOpen(1, mkHash(201), 512, flow.MiB)
	s.callbackOpen(2, mkHash(202), 512, flow.MiB)
	foreground, warm := s.lookup(1), s.lookup(2)
	r := NewReader(foreground, nil, FileInfo{Index: 1, Length: 512 * flow.MiB}, "phone")
	defer r.Close()
	foreground.SetPreloadReserve([][2]int{{0, 180}, {240, 255}})
	data := bytes.Repeat([]byte{37}, int(flow.MiB))
	if _, err := foreground.writePiece(0, 0, data); err != nil {
		t.Fatal(err)
	}
	foreground.SignalPieceComplete(0)
	observe := func(stage string) {
		t.Helper()
		if _, err := r.Seek(0, io.SeekStart); err != nil {
			t.Fatal(err)
		}
		buf := make([]byte, len(data))
		started := time.Now()
		if _, err := io.ReadFull(r, buf); err != nil || !bytes.Equal(buf, data) {
			t.Fatal("protected read failed", err)
		}
		elapsed := time.Since(started)
		foreground.mu.RLock()
		evictions := len(foreground.evicted)
		foreground.mu.RUnlock()
		encoded, _ := json.Marshal(map[string]any{"stage": stage, "injected_pressure": true, "resources": Resources(), "memory": diagnostics.Memory(), "read_elapsed_ms": float64(elapsed) / float64(time.Millisecond), "read_waits": foreground.FlowCounters(), "recent_eviction_markers": evictions})
		t.Log(string(encoded))
	}
	warm.SetWarmReserve(300, 340, time.Minute)
	now := time.Now()
	s.reconcileResources(diagnostics.MemoryStatus{PressureState: "normal"}, now)
	observe("before")
	s.reconcileResources(diagnostics.MemoryStatus{Pressure: true, PressureState: "pressure"}, now.Add(5*time.Second))
	if Resources().WarmBudgetBytes != 0 || !warm.backgroundLimited.Load() || foreground.capacity() < 197*flow.MiB || !Resources().ProtectedOvercommit {
		t.Fatal("pressure removed demand protection or retained background", Resources())
	}
	warm.preloadMu.Lock()
	retained := !warm.warmUntil.IsZero()
	warm.preloadMu.Unlock()
	if retained {
		t.Fatal("warm lease not released")
	}
	observe("pressure")
	for i := 2; i < 12; i++ {
		s.reconcileResources(diagnostics.MemoryStatus{Pressure: true, PressureState: "recovering", RecoverySamples: 2}, now.Add(time.Duration(i)*5*time.Second))
	}
	if Resources().BudgetChanges != 2 {
		t.Fatal("recovering samples churned cache budgets", Resources())
	}
	observe("recovering")
	s.reconcileResources(diagnostics.MemoryStatus{PressureState: "normal"}, now.Add(time.Minute))
	if warm.backgroundLimited.Load() || foreground.backgroundLimited.Load() || Resources().BudgetChanges != 3 || Resources().ProtectedOvercommit {
		t.Fatal("background did not recover", Resources())
	}
	observe("recovered")
}

func TestPreloadGrowthSharesCacheAndResourceBudgets(t *testing.T) {
	old := settings.BTsets()
	t.Cleanup(func() { settings.StoreBTsets(old) })
	settings.StoreBTsets(&settings.BTSets{CacheSize: 128 << 20, Flow: settings.DefaultFlowSettings()})
	c := &Cache{PieceLength: 16 << 20}
	if got := c.PreloadHeadBudget(); got != 96<<20 {
		t.Fatal("head did not leave room for tail and file boundary", got)
	}
	settings.StoreBTsets(&settings.BTSets{CacheSize: 100 << 20, Flow: settings.DefaultFlowSettings()})
	if got := c.PreloadHeadBudget(); got != 64<<20 {
		t.Fatal("fractional-piece cache budget was rounded up", got)
	}
	c.resourceBudget.Store(64 << 20)
	if got := c.PreloadHeadBudget(); got != 32<<20 {
		t.Fatal("resource budget ignored", got)
	}
	c.resourceBudget.Store(16 << 20)
	if got := c.PreloadHeadBudget(); got != 0 {
		t.Fatal("small cache allowed adaptive growth", got)
	}
}

func TestGlobalBudgetProtectsReadersAndReleasesWarmReservations(t *testing.T) {
	old := settings.BTsets()
	f := settings.DefaultFlowSettings()
	f.GlobalCacheBudgetMB = 64
	f.WarmCacheBudgetMB = 0
	settings.StoreBTsets(&settings.BTSets{CacheSize: 512 * flow.MiB, ReaderReadAHead: 95, Flow: f})
	t.Cleanup(func() {
		settings.StoreBTsets(old)
		resourceState.Lock()
		resourceState.status = ResourceStatus{}
		resourceState.Unlock()
	})
	s := NewStorage()
	for i := int64(1); i <= 3; i++ {
		s.callbackOpen(i, mkHash(byte(i+20)), 256, flow.MiB)
	}
	first, second, warm := s.lookup(1), s.lookup(2), s.lookup(3)
	r1 := NewReader(first, nil, FileInfo{Index: 1, Length: 256 * flow.MiB}, "one")
	defer r1.Close()
	r2 := NewReader(second, nil, FileInfo{Index: 1, Length: 256 * flow.MiB}, "two")
	defer r2.Close()
	warm.SetWarmReserve(40, 60, time.Minute)
	resourceState.Lock()
	resourceState.status = ResourceStatus{}
	resourceState.Unlock()
	s.ReconcileResources()
	if first.resourceBudget.Load()+second.resourceBudget.Load() > 64*flow.MiB || first.resourceBudget.Load() == 0 {
		t.Fatal("active shares exceed budget")
	}
	warm.preloadMu.Lock()
	retained := !warm.warmUntil.IsZero()
	warm.preloadMu.Unlock()
	if retained {
		t.Fatal("disabled warm budget retained reservation")
	}
	// Mandatory immediate work remains protected even when it exceeds a share.
	first.SetPreloadReserve([][2]int{{0, 80}})
	if first.capacity() < 81*flow.MiB {
		t.Fatal("budget discarded protected preload")
	}
	resourceState.Lock()
	resourceState.status.SampledAt = time.Time{}
	resourceState.Unlock()
	s.ReconcileResources()
	if !Resources().ProtectedOvercommit || !Resources().BackgroundLimited {
		t.Fatal("overcommit was hidden")
	}
}

func TestVerifiedDeliveryCountsOnceAndResetsForReconnect(t *testing.T) {
	old := settings.BTsets()
	settings.StoreBTsets(&settings.BTSets{CacheSize: 64 * flow.MiB, ReaderReadAHead: 70, Flow: settings.DefaultFlowSettings()})
	t.Cleanup(func() { settings.StoreBTsets(old) })
	s := NewStorage()
	s.callbackOpen(8, mkHash(88), 100, flow.MiB)
	c := s.lookup(8)
	r := NewReader(c, nil, FileInfo{Index: 1, Length: 100 * flow.MiB}, "phone")
	defer r.Close()
	r.offset.Store(20 * flow.MiB)
	r.winFirst.Store(20)
	r.winLast.Store(60)
	r.lastRead.Store(time.Now().Unix())
	c.SetFlowMediaEstimate("phone", 1, flow.Estimate{BytesPerSecond: 1024, Confidence: "high"})
	c.SetFlowDownloadRate(0, false)
	_, _ = s.callbackWrite(8, 20, 0, bytes.Repeat([]byte{1}, int(flow.MiB)))
	c.SignalPieceComplete(20)
	c.SignalPieceComplete(20)
	c.flowMu.Lock()
	e := c.flowGroups["phone"].delivery.Snapshot(time.Now().Add(time.Second))
	c.flowMu.Unlock()
	if e.LongRate != float64(flow.MiB) || e.Samples != 1 {
		t.Fatal("completion not counted exactly once", e)
	}
	s.InvalidateNetworkEvidence()
	if e = c.FlowWindow("phone").Delivery; e.Samples != 0 || c.flowReconnect.Load() {
		t.Fatal("same-address route recovery retained supply or paused playback", e)
	}
	s.SetNetworkRecovering(true)
	s.callbackOpen(9, mkHash(89), 100, flow.MiB)
	if !s.lookup(9).flowReconnect.Load() {
		t.Fatal("cache created during recovery lost network state")
	}
	c.SetFlowDownloadRate(0, false)
	if e = c.FlowWindow("phone").Delivery; e.Mode != "RECONNECT" || e.Confidence != "unknown" {
		t.Fatal(e)
	}
	s.SetNetworkRecovering(false)
	c.SetFlowDownloadRate(0, false)
	if e = c.FlowWindow("phone").Delivery; e.Samples != 0 {
		t.Fatal("old supply survived reconnect", e)
	}
	c.SetFlowDownloadRate(0, true)
	if e = c.FlowWindow("phone").Delivery; e.Mode != "IDLE" || e.Confidence != "unknown" {
		t.Fatal("intentional pause was treated as supply loss", e)
	}
	c.SetFlowDownloadRate(0, false)
	if e = c.FlowWindow("phone").Delivery; e.Samples != 0 {
		t.Fatal("pre-pause supply survived resume", e)
	}
}
