package torrstor

import (
	"math"
	"time"

	"server/flow"
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
}

type FlowWindowStatus struct {
	RecentDownloadRate   float64 `json:"recent_download_rate"`
	DownloadRateSamples  int     `json:"download_rate_samples"`
	ObservedPlaybackRate float64 `json:"observed_playback_rate"`
	ObservedConfidence   string  `json:"observed_confidence"`
	TargetBufferSeconds  int     `json:"target_buffer_seconds"`
	ForwardWindowPieces  int     `json:"forward_window_pieces"`
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
	c.flowMu.Lock()
	now := time.Now()
	if rate >= 0 && !math.IsNaN(rate) && !math.IsInf(rate, 0) {
		c.flowRates.Observe(rate, now)
		c.flowDownload, _ = c.flowRates.Mean(now)
	}
	c.refreshFlowWindowLocked(now)
	c.flowMu.Unlock()
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
		return FlowWindowStatus{RecentDownloadRate: download, DownloadRateSamples: samples}
	}
	rate, confidence := g.tracker.Rate()
	return FlowWindowStatus{ObservedPlaybackRate: rate, ObservedConfidence: confidence,
		RecentDownloadRate: download, DownloadRateSamples: samples,
		TargetBufferSeconds: g.seconds, ForwardWindowPieces: g.pieces}
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
		if observed, _ := g.tracker.Rate(); observed > 0 {
			rate = observed
		}
		g.seconds, g.pieces = flow.AdaptiveWindow(rate, c.flowDownload, waitP95,
			f.TargetBufferSeconds, f.MaxBufferSeconds, f.StartupSafetyFactorPct,
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
		c.refreshFlowWindowLocked(time.Now())
	}
	c.flowMu.Unlock()
}
