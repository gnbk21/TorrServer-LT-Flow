package torrstor

import (
	"bytes"
	"server/flow"
	"server/settings"
	"testing"
	"time"
)

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
	c.SetFlowDownloadRate(0)
	_, _ = s.callbackWrite(8, 20, 0, bytes.Repeat([]byte{1}, flow.MiB))
	c.SignalPieceComplete(20)
	c.SignalPieceComplete(20)
	c.flowMu.Lock()
	e := c.flowGroups["phone"].delivery.Snapshot(time.Now().Add(time.Second))
	c.flowMu.Unlock()
	if e.LongRate != flow.MiB || e.Samples != 1 {
		t.Fatal("completion not counted exactly once", e)
	}
	s.SetNetworkRecovering(true)
	c.SetFlowDownloadRate(0)
	if e = c.FlowWindow("phone").Delivery; e.Mode != "RECONNECT" || e.Confidence != "unknown" {
		t.Fatal(e)
	}
	s.SetNetworkRecovering(false)
	c.SetFlowDownloadRate(0)
	if e = c.FlowWindow("phone").Delivery; e.Samples != 0 {
		t.Fatal("old supply survived reconnect", e)
	}
}
