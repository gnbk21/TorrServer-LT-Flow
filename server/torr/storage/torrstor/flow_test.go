package torrstor

import (
	"bytes"
	"context"
	"testing"
	"time"

	"server/flow"
	"server/lt"
	"server/settings"
)

func TestDemandRatesRequireExplicitExperimentAndQualifiedFreshMedia(t *testing.T) {
	old := settings.BTsets()
	f := settings.DefaultFlowSettings()
	settings.StoreBTsets(&settings.BTSets{Flow: f})
	t.Cleanup(func() { settings.StoreBTsets(old) })
	c := &Cache{flowGroups: map[string]*flowGroup{
		"fresh": {seen: time.Now(), estimate: flow.Estimate{BytesPerSecond: 1024, Confidence: "high"}},
		"weak":  {seen: time.Now(), estimate: flow.Estimate{BytesPerSecond: 1024, Confidence: "low"}},
		"old":   {seen: time.Now().Add(-31 * time.Second), estimate: flow.Estimate{BytesPerSecond: 1024, Confidence: "high"}},
	}}
	if len(c.demandRates()) != 0 {
		t.Fatal("default policy changed deadline timing")
	}
	enabled := *f
	enabled.RateAwareDeadlines = true
	settings.StoreBTsets(&settings.BTSets{Flow: &enabled})
	rates := c.demandRates()
	if len(rates) != 1 || rates["fresh"] != 1024 {
		t.Fatal("unqualified or stale rate selected", rates)
	}
	scarce := *f
	scarce.ScarcePieceHints = true
	settings.StoreBTsets(&settings.BTSets{Flow: &scarce})
	if c.demandRates()["fresh"] != 1024 {
		t.Fatal("independent scarce experiment lost its qualification evidence")
	}
}

func TestUrgentHorizonsPreserveIndependentReadersAndRejectStaleQueue(t *testing.T) {
	old := settings.BTsets()
	f := settings.DefaultFlowSettings()
	f.AdaptiveUrgentHorizon = true
	settings.StoreBTsets(&settings.BTSets{Flow: f})
	t.Cleanup(func() { settings.StoreBTsets(old) })
	c := &Cache{PieceLength: 4 * flow.MiB, flowGroups: map[string]*flowGroup{
		"slow": {seen: time.Now(), estimate: flow.Estimate{BytesPerSecond: float64(flow.MiB), Confidence: "high"}},
		"fast": {seen: time.Now(), estimate: flow.Estimate{BytesPerSecond: 15e6, Confidence: "high"}},
	}}
	sample := lt.SparseSnapshot{Known: true, SampledAtMs: time.Now().UnixMilli(), MaxQueueMs: 1000}
	c.SetScarceEvidence(sample)
	horizons := c.urgentHorizons(c.demandRates(), 100)
	if horizons["slow"] != 3 || horizons["fast"] != 10 {
		t.Fatal("independent demand collapsed", horizons)
	}
	sample.Truncated = true
	c.SetScarceEvidence(sample)
	if len(c.urgentHorizons(c.demandRates(), 100)) != 0 {
		t.Fatal("truncated evidence changed deadline policy")
	}
	sample.Truncated = false
	sample.SampledAtMs = time.Now().Add(-6 * time.Second).UnixMilli()
	c.SetScarceEvidence(sample)
	if len(c.urgentHorizons(c.demandRates(), 100)) != 0 {
		t.Fatal("stale queue changed deadline policy")
	}
}

// Deadline removal changes the native priority independently of lastPrios.
// Exercise the actual shim, not just the Go vector comparison.
func TestDeadlineRemovalRestoresLazyAndReservedNativePriorities(t *testing.T) {
	for _, reserved := range []bool{false, true} {
		session, err := lt.NewSession(lt.SessionConfig{"enable_dht": false, "enable_lsd": false, "enable_upnp": false, "enable_natpmp": false, "listen_interfaces": "127.0.0.1:0"})
		if err != nil {
			t.Fatal(err)
		}
		func() {
			defer session.Close()
			info := append([]byte("d4:infod6:lengthi100e4:name4:test12:piece lengthi16384e6:pieces20:"), bytes.Repeat([]byte{0}, 20)...)
			info = append(info, 'e', 'e')
			handle, err := session.AddTorrent(lt.AddTorrentParams{InfoBytes: info, SavePath: t.TempDir(), Paused: true})
			if err != nil {
				t.Fatal(err)
			}
			defer handle.Remove(false)
			want := 0
			c := &Cache{NumPieces: 1, PieceLength: 16384, deadlined: map[int]bool{0: true}, refetchedAt: map[int]int64{}}
			if reserved {
				want = streamPreloadPriority
				c.preloadProtect = [][2]int{{0, 0}}
			}
			c.lastPrios = []int{want}
			c.handle.Store(handle)
			if err := handle.SetPieceDeadline(0, 0, false); err != nil {
				t.Fatal(err)
			}
			c.applyStreamPriorities()
			deadline := time.Now().Add(4 * time.Second)
			for time.Now().Before(deadline) {
				snapshot, err := handle.SampleSparse([][2]int{{0, 1}})
				if err != nil {
					t.Fatal(err)
				}
				if snapshot.Known && len(snapshot.Urgent) == 1 {
					if snapshot.Urgent[0].Priority != want {
						t.Fatalf("reserved=%v: native priority %d, want %d", reserved, snapshot.Urgent[0].Priority, want)
					}
					return
				}
				time.Sleep(20 * time.Millisecond)
			}
			t.Fatal("native priority unavailable")
		}()
	}
}

func TestScarceEvidenceRejectsStaleTruncatedAndUnqualifiedDelivery(t *testing.T) {
	old := settings.BTsets()
	f := settings.DefaultFlowSettings()
	f.ScarcePieceHints = true
	settings.StoreBTsets(&settings.BTSets{Flow: f})
	t.Cleanup(func() { settings.StoreBTsets(old) })
	c := &Cache{flowGroups: map[string]*flowGroup{"phone": {}}}
	for i := 0; i < 6; i++ {
		at := time.Now().Add(time.Duration(i-6) * time.Second)
		c.flowGroups["phone"].delivery.Observe(at, "DEMAND")
		c.flowGroups["phone"].delivery.AddVerified(int64(1+i%2*20)*flow.MiB, at)
	}
	c.flowGroups["phone"].delivery.Observe(time.Now(), "DEMAND")
	sample := lt.SparseSnapshot{Known: true, SampledAtMs: time.Now().UnixMilli(), Windows: []lt.SparseWindow{{FirstPiece: 10, Availability: []int{2, 1, 0}}}}
	c.SetScarceEvidence(sample)
	hints, variable := c.scarceDemand()
	if !variable || !hints[11] || hints[10] || hints[12] {
		t.Fatalf("bad scarce evidence: %v %v", hints, variable)
	}
	sample.Truncated = true
	c.SetScarceEvidence(sample)
	if hints, _ := c.scarceDemand(); len(hints) != 0 {
		t.Fatal("partial sample used as sole-supplier proof")
	}
	sample.Truncated = false
	sample.SampledAtMs = time.Now().Add(-6 * time.Second).UnixMilli()
	c.SetScarceEvidence(sample)
	if hints, _ := c.scarceDemand(); len(hints) != 0 {
		t.Fatal("stale sample changed demand")
	}
}

func TestParkedReadReconcilesLateCompletionAndCancels(t *testing.T) {
	old := settings.BTsets()
	settings.StoreBTsets(&settings.BTSets{CacheSize: 64 * flow.MiB})
	t.Cleanup(func() { settings.StoreBTsets(old) })
	s := NewStorage()
	h := mkHash(0xE3)
	s.callbackOpen(3, h, 4, pieceBlockSize)
	c := s.CacheByHash(h)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	count := 0
	if !c.waitForBytes(ctx, 0, 0, func() { count++; _, _ = s.callbackWrite(3, 0, 0, bytes.Repeat([]byte{7}, pieceBlockSize)) }) || count != 1 {
		t.Fatal("parked read did not reconcile missing data")
	}
	ctx, cancel = context.WithCancel(context.Background())
	cancel()
	if c.waitForBytes(ctx, 1, 0, func() { t.Fatal("reconciled cancelled request") }) {
		t.Fatal("cancelled reader continued waiting")
	}
}

func TestPartialPruneRechecksActivityAndStoresLateBlocks(t *testing.T) {
	old := settings.BTsets()
	settings.StoreBTsets(&settings.BTSets{CacheSize: 64 * flow.MiB})
	t.Cleanup(func() { settings.StoreBTsets(old) })
	s := NewStorage()
	s.callbackOpen(31, mkHash(0xD7), 128, 4*pieceBlockSize)
	c := s.lookup(31)
	block := bytes.Repeat([]byte{0x73}, pieceBlockSize)
	_, _ = s.callbackWrite(31, 50, 0, block)
	p := c.pieces[50]
	if s.callbackPrune(31, 50) {
		t.Fatal("recent block was discarded")
	}
	p.accessed.Store(time.Now().Unix() - abandonEvictSec - 2)
	r := NewReader(c, nil, FileInfo{Offset: 50 * c.PieceLength, Length: c.PieceLength}, "phone")
	if s.callbackPrune(31, 50) {
		t.Fatal("new reader's target was discarded")
	}
	_ = r.Close()
	// Drop the closed reader's warm reservation for this isolated prune check.
	c.groupsMu.Lock()
	c.groups = map[string]*group{}
	c.groupsMu.Unlock()
	if !s.callbackPrune(31, 50) || !c.consumeEvicted(50) {
		t.Fatal("settled stale partial was not discarded and marked")
	}
	_, _ = s.callbackWrite(31, 50, pieceBlockSize, block)
	dst := make([]byte, pieceBlockSize)
	if n, err := s.callbackRead(31, 50, pieceBlockSize, dst); err != nil || n != len(dst) || !bytes.Equal(dst, block) {
		t.Fatal("late native block was acknowledged without retaining its bytes")
	}
	if c.readableAt(50, 0) != 0 {
		t.Fatal("unwritten earlier block became readable")
	}
}

func TestCompletionAlertCannotCompleteReplacementWithHoles(t *testing.T) {
	s := NewStorage()
	s.callbackOpen(32, mkHash(0xD8), 2, 4*pieceBlockSize)
	s.callbackSize(32, 5*pieceBlockSize+123)
	c := s.lookup(32)
	block := bytes.Repeat([]byte{0x74}, pieceBlockSize)
	// A delayed completion from the previous incarnation arrives after only
	// the last block of the replacement buffer has been downloaded.
	_, _ = s.callbackWrite(32, 0, 3*pieceBlockSize, block)
	c.SignalPieceComplete(0)
	if c.Have(0) || c.readableAt(0, 0) != 0 {
		t.Fatal("stale alert completed a replacement buffer containing holes")
	}
	for b := 0; b < 3; b++ {
		_, _ = s.callbackWrite(32, 0, int64(b*pieceBlockSize), block)
	}
	c.SignalPieceComplete(0)
	if !c.Have(0) {
		t.Fatal("resident verified piece was not completed")
	}
	// The final piece has one full block and a short block. Knowing its exact
	// length must neither expose zero-filled padding nor suppress completion.
	_, _ = s.callbackWrite(32, 1, pieceBlockSize, block[:123])
	c.SignalPieceComplete(1)
	if c.Have(1) || c.readableAt(1, 0) != 0 {
		t.Fatal("short final piece with a hole was completed")
	}
	_, _ = s.callbackWrite(32, 1, 0, block)
	c.SignalPieceComplete(1)
	if !c.Have(1) || c.readableAt(1, 0) != pieceBlockSize+123 || c.readableAt(1, pieceBlockSize+123) != 0 {
		t.Fatal("short final piece availability does not match metadata")
	}
}

func TestContiguousAvailableStopsAtPartialPieceHole(t *testing.T) {
	old := settings.BTsets()
	settings.StoreBTsets(&settings.BTSets{CacheSize: 64 * flow.MiB})
	t.Cleanup(func() { settings.StoreBTsets(old) })
	s := NewStorage()
	h := mkHash(0xE1)
	s.callbackOpen(1, h, 4, 4*pieceBlockSize)
	c := s.CacheByHash(h)
	if c == nil {
		t.Fatal("cache missing")
	}
	block := bytes.Repeat([]byte{1}, pieceBlockSize)
	if _, err := s.callbackWrite(1, 0, 0, block); err != nil {
		t.Fatal(err)
	}
	if got := c.ContiguousAvailable(0, 3*pieceBlockSize); got != pieceBlockSize {
		t.Fatalf("first block: got %d", got)
	}
	if got := c.VerifiedContiguousAvailable(0, 3*pieceBlockSize); got != 0 {
		t.Fatal("unverified blocks counted as verified reserve", got)
	}
	if _, err := s.callbackWrite(1, 0, 2*pieceBlockSize, block); err != nil {
		t.Fatal(err)
	}
	if got := c.ContiguousAvailable(0, 3*pieceBlockSize); got != pieceBlockSize {
		t.Fatalf("hole should stop run: got %d", got)
	}
	if _, err := s.callbackWrite(1, 0, pieceBlockSize, block); err != nil {
		t.Fatal(err)
	}
	if got := c.ContiguousAvailable(0, 3*pieceBlockSize); got != 3*pieceBlockSize {
		t.Fatalf("contiguous blocks: got %d", got)
	}
}

func TestAdaptiveWindowUsesExistingCacheBudget(t *testing.T) {
	old := settings.BTsets()
	settings.StoreBTsets(&settings.BTSets{CacheSize: 512 * flow.MiB, ReaderReadAHead: 95, Flow: settings.DefaultFlowSettings()})
	t.Cleanup(func() { settings.StoreBTsets(old) })
	s := NewStorage()
	h := mkHash(0xE2)
	s.callbackOpen(2, h, 256, 4*flow.MiB)
	c := s.CacheByHash(h)
	if c == nil {
		t.Fatal("cache missing")
	}
	_, maxAhead := c.baseReaderWindowPieces()
	c.SetFlowMediaEstimate("phone", 0, flow.Estimate{BytesPerSecond: 2 * float64(flow.MiB)})
	c.SetFlowDownloadRate(10*float64(flow.MiB), false)
	_, healthy := c.readerWindowPieces()
	if healthy <= 0 || healthy >= maxAhead {
		t.Fatalf("adaptive ahead=%d, budget maximum=%d", healthy, maxAhead)
	}
	c.SetFlowDownloadRate(float64(flow.MiB)/2, false)
	_, unqualified := c.readerWindowPieces()
	if unqualified != healthy {
		t.Fatal("aggregate wire rate changed qualified controller", unqualified, healthy)
	}
	c.flowMu.Lock()
	g := c.flowGroups["phone"]
	now := time.Now()
	for i := 6; i > 0; i-- {
		at := now.Add(-time.Duration(i) * time.Second)
		g.delivery.Observe(at, "DEMAND")
		g.delivery.AddVerified(flow.MiB/2, at)
	}
	g.delivery.Observe(now, "DEMAND")
	c.refreshFlowWindowLocked(now)
	c.flowMu.Unlock()
	_, weak := c.readerWindowPieces()
	if weak <= healthy || weak > maxAhead {
		t.Fatalf("weak swarm ahead=%d, healthy=%d, max=%d", weak, healthy, maxAhead)
	}
}

func TestAdaptiveProfileUsesBoundedBurstHintAndImmediateBudgetReduction(t *testing.T) {
	old := settings.BTsets()
	f := settings.DefaultFlowSettings()
	settings.StoreBTsets(&settings.BTSets{CacheSize: 512 * flow.MiB, ReaderReadAHead: 95, Flow: f})
	t.Cleanup(func() { settings.StoreBTsets(old) })
	s := NewStorage()
	s.callbackOpen(22, mkHash(0xE3), 2048, flow.MiB)
	c := s.lookup(22)
	c.SetFlowMediaEstimate("phone", 0, flow.Estimate{BytesPerSecond: float64(flow.MiB), Confidence: "high"})
	now := time.Now()
	c.flowMu.Lock()
	g := c.flowGroups["phone"]
	for i := 0; i < 6; i++ {
		g.tracker.Observe(int64(i)*4*flow.MiB, now.Add(time.Duration(i-5)*2*time.Second), float64(flow.MiB))
	}
	c.refreshFlowWindowLocked(now)
	legacy := g.pieces
	g.smoother.Reset()
	c.flowMu.Unlock()
	adaptive := *f
	adaptive.SwarmProfile = "adaptive"
	settings.StoreBTsets(&settings.BTSets{CacheSize: 512 * flow.MiB, ReaderReadAHead: 95, Flow: &adaptive})
	c.flowMu.Lock()
	c.refreshFlowWindowLocked(now)
	burst := g.pieces
	c.flowMu.Unlock()
	_, maximum := c.baseReaderWindowPieces()
	if burst <= legacy || burst > maximum {
		t.Fatalf("adaptive=%d legacy=%d budget=%d", burst, legacy, maximum)
	}
	settings.StoreBTsets(&settings.BTSets{CacheSize: 64 * flow.MiB, ReaderReadAHead: 95, Flow: &adaptive})
	c.flowMu.Lock()
	c.refreshFlowWindowLocked(now.Add(time.Second))
	limited := g.pieces
	c.flowMu.Unlock()
	_, maximum = c.baseReaderWindowPieces()
	if limited > maximum || limited >= burst {
		t.Fatalf("reduced budget ignored: pieces=%d maximum=%d before=%d", limited, maximum, burst)
	}
}

func TestFullDeliveryWindowRequiresContiguousReadableBytes(t *testing.T) {
	old := settings.BTsets()
	f := settings.DefaultFlowSettings()
	f.RateAwareDeadlines = true // This test also exercises qualified experiment rates.
	settings.StoreBTsets(&settings.BTSets{CacheSize: 8 * flow.MiB, ReaderReadAHead: 70, Flow: f})
	t.Cleanup(func() { settings.StoreBTsets(old) })
	s := NewStorage()
	s.callbackOpen(45, mkHash(0xEA), 256, 64<<10)
	s.callbackSize(45, 256*(64<<10))
	c := s.lookup(45)
	r := NewReader(c, nil, FileInfo{Index: 1, Offset: 0, Length: 256 * (64 << 10)}, "phone")
	defer r.Close()
	r.offset.Store(64 * c.PieceLength)
	r.winFirst.Store(64)
	r.winLast.Store(100)
	r.lastRead.Store(time.Now().Unix())
	anchors := map[string]int{"phone": 64}
	if c.deliveryWindowsFull(anchors) {
		t.Fatal("missing window reported full")
	}
	_, ahead := c.readerWindowPieces()
	payload := bytes.Repeat([]byte{0x51}, int(c.PieceLength))
	for piece := 64; piece <= 64+ahead; piece++ {
		if _, err := s.callbackWrite(45, piece, 0, payload); err != nil {
			t.Fatal(err)
		}
		c.MarkComplete(piece)
	}
	if !c.deliveryWindowsFull(anchors) {
		t.Fatal("contiguous full window reported missing")
	}
	c.flowWaiting.Store(1)
	if c.deliveryWindowsFull(anchors) {
		t.Fatal("blocked demand treated as idle full buffer")
	}
	c.SetFlowMediaEstimate("phone", 1, flow.Estimate{BytesPerSecond: float64(flow.MiB), Confidence: "low"})
	if c.demandRates()["phone"] != 0 {
		t.Fatal("provisional demand rate used for deadlines")
	}
	c.SetFlowMediaEstimate("phone", 1, flow.Estimate{BytesPerSecond: float64(flow.MiB), Confidence: "medium"})
	if c.demandRates()["phone"] != float64(flow.MiB) {
		t.Fatal("qualified metadata rate ignored")
	}
}
