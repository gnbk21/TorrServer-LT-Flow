package torrstor

import (
	"math"
	"time"

	"server/flow"
	"server/lt"
	"server/settings"
)

type flowGroup struct {
	fileIndex int
	estimate  flow.Estimate
	tracker   flow.ConsumptionTracker
	smoother  flow.WindowSmoother
	seen      time.Time
	seconds   int
	pieces    int
	delivery  flow.DeliveryMeter
	risk      flow.RiskDecision
	buffer    int64
	anchor    int
	full      bool
}

type FlowWindowStatus struct {
	Delivery             flow.DeliveryEvidence `json:"delivery"`
	Risk                 flow.RiskDecision     `json:"risk"`
	RecentDownloadRate   float64               `json:"recent_download_rate"`
	DownloadRateSamples  int                   `json:"download_rate_samples"`
	ObservedPlaybackRate float64               `json:"observed_playback_rate"`
	ObservedConfidence   string                `json:"observed_confidence"`
	TargetBufferSeconds  int                   `json:"target_buffer_seconds"`
	ForwardWindowPieces  int                   `json:"forward_window_pieces"`
}

func (c *Cache) SetScarceEvidence(snapshot lt.SparseSnapshot) {
	if c == nil {
		return
	}
	c.flowMu.Lock()
	defer c.flowMu.Unlock()
	c.flowScarce = nil
	c.flowSuppliers = nil
	c.flowScarceAt = time.Time{}
	age := time.Since(time.UnixMilli(snapshot.SampledAtMs))
	if !snapshot.Known || snapshot.Truncated || age < 0 || age > 5*time.Second {
		return
	}
	c.flowScarce = make(map[int]bool)
	c.flowSuppliers = make(map[int]int)
	for _, window := range snapshot.Windows {
		for i, count := range window.Availability {
			if len(c.flowSuppliers) < 256 {
				c.flowSuppliers[window.FirstPiece+i] = count
			}
			if count == 1 && len(c.flowScarce) < 256 {
				c.flowScarce[window.FirstPiece+i] = true
			}
		}
	}
	c.flowScarceAt = time.UnixMilli(snapshot.SampledAtMs)
}

func (c *Cache) scarceDemand() (map[int]bool, bool) {
	if !settings.CurrentFlow().ScarcePieceHints {
		return nil, false
	}
	c.flowMu.Lock()
	defer c.flowMu.Unlock()
	age := time.Since(c.flowScarceAt)
	variable := false
	for _, g := range c.flowGroups {
		e := g.delivery.Snapshot(time.Now())
		variable = variable || (e.Confidence != "unknown" && e.Samples >= 5 && e.Variation > .5)
	}
	if age < 0 || age > 5*time.Second || !variable {
		return nil, false
	}
	out := make(map[int]bool, len(c.flowScarce))
	for piece, sole := range c.flowScarce {
		out[piece] = sole
	}
	return out, true
}

func (c *Cache) demandRates() map[string]float64 {
	rates := make(map[string]float64)
	f := settings.CurrentFlow()
	if !f.Enabled || !f.AdaptiveReadAhead || (!f.RateAwareDeadlines && !f.ScarcePieceHints) {
		return rates
	}
	c.flowMu.Lock()
	defer c.flowMu.Unlock()
	for group, g := range c.flowGroups {
		if time.Since(g.seen) > 30*time.Second {
			continue
		}
		if g.estimate.Confidence == "medium" || g.estimate.Confidence == "high" {
			rates[group] = g.estimate.BytesPerSecond
		} else if observed, confidence := g.tracker.Rate(); observed > 0 && confidence == "stable" {
			rates[group] = observed
		}
	}
	return rates
}

func (c *Cache) SetFlowMediaEstimate(group string, fileIndex int, estimate flow.Estimate) {
	if c == nil || group == ProbeReaderGroup || !settings.CurrentFlow().Enabled {
		return
	}
	c.flowMu.Lock()
	if c.flowGroups == nil {
		c.flowGroups = make(map[string]*flowGroup)
	}
	g := c.flowGroups[group]
	if g == nil || g.fileIndex != fileIndex {
		if g == nil && len(c.flowGroups) >= 128 {
			var oldestKey string
			var oldestAt time.Time
			for key, candidate := range c.flowGroups {
				if oldestKey == "" || candidate.seen.Before(oldestAt) {
					oldestKey, oldestAt = key, candidate.seen
				}
			}
			delete(c.flowGroups, oldestKey)
		}
		g = &flowGroup{fileIndex: fileIndex}
		c.flowGroups[group] = g
	}
	g.estimate, g.seen = estimate, time.Now()
	c.refreshFlowWindowLocked(g.seen)
	c.flowMu.Unlock()
}

func (c *Cache) UpdateFlowMediaEstimate(fileIndex int, estimate flow.Estimate) {
	if c == nil || !settings.CurrentFlow().Enabled {
		return
	}
	c.flowMu.Lock()
	for _, g := range c.flowGroups {
		if g.fileIndex == fileIndex {
			g.estimate = estimate
		}
	}
	c.refreshFlowWindowLocked(time.Now())
	c.flowMu.Unlock()
}

func (c *Cache) ObserveFlowProgress(group string, fileIndex int, offset int64) {
	if c == nil || group == ProbeReaderGroup || !settings.CurrentFlow().Enabled {
		return
	}
	c.flowMu.Lock()
	g := c.flowGroups[group]
	if g != nil && g.fileIndex == fileIndex {
		now := time.Now()
		g.tracker.Observe(offset, now, g.estimate.BytesPerSecond)
		g.seen = now
		// Rates need seconds of observations; rebuilding a whole window on every
		// 16 KiB HTTP Read would only add lock/cgo pressure.
		if now.Sub(c.flowLastRefresh) >= time.Second {
			c.refreshFlowWindowLocked(now)
		}
	}
	c.flowMu.Unlock()
}

func (c *Cache) SetFlowDownloadRate(rate float64) {
	if c == nil || !settings.CurrentFlow().Enabled {
		return
	}
	if c.storage != nil {
		c.storage.ReconcileResources()
	}
	// Sample the existing reader windows before flowMu: streamAnchors acquires
	// reader/group locks and must not be called during controller reconciliation.
	anchors := c.streamAnchors()
	c.flowBufferFull.Store(c.deliveryWindowsFull(anchors))
	snaps := c.groupReaderSnaps()
	preparing := len(c.preparationDemand()) > 0
	probing := len(snaps[ProbeReaderGroup]) > 0
	type sample struct {
		mode   string
		buffer int64
		anchor int
		full   bool
	}
	samples := make(map[string]sample)
	_, ahead := c.readerWindowPieces()
	for group, first := range anchors {
		for _, s := range snaps[group] {
			if s.internal || s.isProbe || s.isFocus || s.stale || s.fileEnd <= s.fileStart {
				continue
			}
			start := max(int64(first)*c.PieceLength, s.fileStart)
			end := min(int64(min(c.NumPieces, first+ahead+1))*c.PieceLength, s.fileEnd)
			if end <= start {
				continue
			}
			buffer := c.ContiguousAvailable(start, end)
			mode := "DEMAND"
			full := buffer == end-start
			if full {
				mode = "FULL"
			}
			samples[group] = sample{mode, buffer, first, full}
			break
		}
	}
	c.flowMu.Lock()
	now := time.Now()
	for group, g := range c.flowGroups {
		mode := "IDLE"
		if preparing {
			mode = "PREPARATION"
		}
		if probing {
			mode = "PROBE"
		}
		if s, ok := samples[group]; ok {
			mode = s.mode
			g.buffer, g.anchor, g.full = s.buffer, s.anchor, s.full
		}
		if group == ProbeReaderGroup {
			mode = "PROBE"
		}
		if c.flowReconnect.Load() {
			mode = "RECONNECT"
		}
		g.delivery.Observe(now, mode)
	}
	for group := range anchors {
		if g := c.flowGroups[group]; g != nil {
			g.seen = now
		}
	}
	if rate >= 0 && !math.IsNaN(rate) && !math.IsInf(rate, 0) {
		c.flowRates.Observe(rate, now)
		c.flowDownload, _ = c.flowRates.Mean(now)
	}
	c.refreshFlowWindowLocked(now)
	c.flowMu.Unlock()
}

func (c *Cache) deliveryWindowsFull(anchors map[string]int) bool {
	if c.flowWaiting.Load() > 0 || c.PieceLength <= 0 {
		return false
	}
	if len(anchors) == 0 {
		return false
	}
	snaps := c.groupReaderSnaps()
	_, ahead := c.readerWindowPieces()
	for group, first := range anchors {
		last := min(c.NumPieces-1, first+ahead)
		for _, s := range snaps[group] {
			if s.cur == first {
				last = min(last, s.flast)
			}
		}
		start, end := int64(first)*c.PieceLength, int64(last+1)*c.PieceLength
		if total := c.totalSize.Load(); total > 0 {
			end = min(end, total)
		}
		if start < 0 || end <= start || c.ContiguousAvailable(start, end) != end-start {
			return false
		}
	}
	return true
}

func (c *Cache) FlowWindow(group string) FlowWindowStatus {
	if c == nil {
		return FlowWindowStatus{}
	}
	c.flowMu.Lock()
	defer c.flowMu.Unlock()
	g := c.flowGroups[group]
	download, samples := c.flowRates.Mean(time.Now())
	if g == nil {
		mode := "IDLE"
		if group == ProbeReaderGroup {
			mode = "PROBE"
		}
		if c.flowReconnect.Load() {
			mode = "RECONNECT"
		}
		return FlowWindowStatus{RecentDownloadRate: download, DownloadRateSamples: samples,
			Delivery: flow.DeliveryEvidence{Mode: mode, Confidence: "unknown", AgeMs: -1},
			Risk:     flow.RiskDecision{Level: "UNKNOWN", Reason: "INSUFFICIENT_EVIDENCE", Confidence: "unknown"}}
	}
	rate, confidence := g.tracker.Rate()
	return FlowWindowStatus{ObservedPlaybackRate: rate, ObservedConfidence: confidence,
		Delivery: g.delivery.Snapshot(time.Now()), Risk: g.risk,
		RecentDownloadRate: download, DownloadRateSamples: samples,
		TargetBufferSeconds: g.seconds, ForwardWindowPieces: g.pieces}
}

func (s *Storage) SetNetworkRecovering(recovering bool) {
	s.networkRecovering.Store(recovering)
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, c := range s.caches {
		if c.flowReconnect.Swap(recovering) == recovering {
			continue
		}
		c.flowMu.Lock()
		for _, g := range c.flowGroups {
			g.delivery.Reset()
			g.smoother.Reset()
		}
		c.flowMu.Unlock()
	}
}

// refreshFlowWindowLocked uses the largest active group's requirement so two
// clients keep distinct anchors while a higher-bitrate file is still protected.
// Call with flowMu held; it never takes readersMu.
func (c *Cache) refreshFlowWindowLocked(now time.Time) {
	c.flowLastRefresh = now
	c.flowDownload, _ = c.flowRates.Mean(now)
	f := settings.CurrentFlow()
	if !f.Enabled || !f.AdaptiveReadAhead {
		c.flowAhead.Store(0)
		return
	}
	_, maxAhead := c.baseReaderWindowPieces()
	maxPieces := 0
	waitP95 := c.flowCounters.Snapshot().RecentPieceWaitP95Ms
	for key, g := range c.flowGroups {
		if now.Sub(g.seen) > time.Duration(f.WarmSessionTimeoutSec)*time.Second {
			delete(c.flowGroups, key)
			continue
		}
		if now.Sub(g.seen) > 30*time.Second {
			continue
		}
		rate := g.estimate.BytesPerSecond
		observed, confidence := g.tracker.Rate()
		if observed > 0 && confidence == "stable" && rate <= 0 {
			rate = observed
		}
		evidence := g.delivery.Snapshot(now)
		stats := flow.DeliveryStats{}
		if evidence.Confidence != "unknown" {
			stats = flow.DeliveryStats{Mean: evidence.LongRate, Samples: evidence.Samples, Variation: evidence.Variation, OutageSeconds: evidence.OutageSeconds}
		}
		var suppliers *int
		if age := now.Sub(c.flowScarceAt); age >= 0 && age <= 5*time.Second {
			if n, ok := c.flowSuppliers[g.anchor]; ok {
				suppliers = &n
			}
		}
		riskRate := rate
		if g.estimate.Confidence != "medium" && g.estimate.Confidence != "high" && confidence != "stable" {
			riskRate = 0
		}
		g.risk = flow.BufferRisk(g.buffer, riskRate, waitP95, suppliers, evidence, f.TargetBufferSeconds, f.MaxBufferSeconds)
		g.seconds, g.pieces = flow.DeliveryWindow(rate, waitP95, stats, c.flowWaiting.Load() > 0, g.full,
			g.risk.TargetSeconds, f.MaxBufferSeconds, f.StartupSafetyFactorPct,
			c.PieceLength, maxAhead)
		g.pieces = g.smoother.Apply(g.pieces, maxAhead, now)
		if g.pieces > maxPieces {
			maxPieces = g.pieces
		}
	}
	c.flowAhead.Store(int64(maxPieces))
}

// ResetFlowWindow is called for a classified seek, never for ServeContent's
// sizing seeks or an older overlapping request finishing behind the playhead.
func (c *Cache) ResetFlowWindow(group string, fileIndex int) {
	if c == nil {
		return
	}
	c.flowMu.Lock()
	if g := c.flowGroups[group]; g != nil && g.fileIndex == fileIndex {
		g.smoother.Reset()
		g.tracker.Reset()
		g.delivery.Reset()
		c.refreshFlowWindowLocked(time.Now())
	}
	c.flowMu.Unlock()
}

// Only a fresh verification within a live group's current forward window counts.
// Probe, background preparation and old-window completions remain wire metrics.
func (c *Cache) observeVerifiedForward(piece int, bytes int64) {
	if c.StreamingReaders() == 0 || !settings.CurrentFlow().Enabled {
		return
	}
	anchors := c.streamAnchors()
	snaps := c.groupReaderSnaps()
	_, ahead := c.readerWindowPieces()
	now := time.Now()
	c.flowMu.Lock()
	defer c.flowMu.Unlock()
	for group, first := range anchors {
		last := min(c.NumPieces-1, first+ahead)
		for _, snap := range snaps[group] {
			if snap.internal || snap.isProbe || snap.isFocus || snap.stale || snap.fileEnd <= snap.fileStart {
				continue
			}
			end := min(last, snap.flast)
			if g := c.flowGroups[group]; g != nil && piece >= first && piece <= end {
				useful := min(int64(piece)*c.PieceLength+bytes, snap.fileEnd) - max(int64(piece)*c.PieceLength, snap.fileStart)
				g.delivery.AddVerified(useful, now)
			}
			break
		}
	}
}
