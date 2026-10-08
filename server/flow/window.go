package flow

import (
	"math"
	"time"
)

// WindowSmoother caps changes in an existing cache window. Growth responds
// immediately, contraction needs ten seconds; tiny fluctuations are ignored.
// Reset for a new file/seek. A reduced hard budget always applies immediately.
type WindowSmoother struct {
	pieces  int
	changed time.Time
}

func (h *WindowSmoother) Reset() { *h = WindowSmoother{} }
func (h *WindowSmoother) Apply(want, maximum int, now time.Time) int {
	want = max(0, min(want, maximum))
	if h.pieces == 0 || h.pieces > maximum {
		h.pieces = want
		h.changed = now
		return want
	}
	delta := want - h.pieces
	step := max(1, h.pieces/4)
	if delta > max(1, h.pieces/10) {
		h.pieces += min(step, delta)
		h.changed = now
	} else if delta < -max(1, h.pieces/10) && now.Sub(h.changed) >= 10*time.Second {
		h.pieces -= min(step, -delta)
		h.changed = now
	}
	return min(maximum, h.pieces)
}

// RateWindow uses one observation per second and expires idle history. Samples
// with no download are retained for reporting but not treated as proof of loss.
type RateWindow struct {
	samples [60]struct {
		second int64
		rate   float64
	}
}

func (r *RateWindow) Observe(rate float64, now time.Time) {
	if rate < 0 || math.IsNaN(rate) || math.IsInf(rate, 0) || now.Unix() <= 0 {
		return
	}
	i := now.Unix() % 60
	r.samples[i].second = now.Unix()
	r.samples[i].rate = rate
}
func (r *RateWindow) Mean(now time.Time) (float64, int) {
	var total float64
	count := 0
	for _, s := range r.samples {
		age := now.Unix() - s.second
		if s.second > 0 && age >= 0 && age < 60 {
			total += s.rate
			count++
		}
	}
	if count == 0 {
		return 0, 0
	}
	return total / float64(count), count
}

type DeliveryStats struct {
	Mean          float64
	Samples       int
	Variation     float64 // coefficient of variation among positive deliveries
	OutageSeconds int     // consecutive observed seconds with zero delivery
}

func (r *RateWindow) Stats(now time.Time) DeliveryStats {
	mean, count := r.Mean(now)
	out := DeliveryStats{Mean: mean, Samples: count}
	var sum, squares float64
	positive := 0
	for _, sample := range r.samples {
		age := now.Unix() - sample.second
		if sample.second > 0 && age >= 0 && age < 60 && sample.rate > 0 {
			sum += sample.rate
			squares += sample.rate * sample.rate
			positive++
		}
	}
	if positive >= 5 {
		average := sum / float64(positive)
		out.Variation = math.Sqrt(math.Max(0, squares/float64(positive)-average*average)) / average
	}
	for age := 0; age < 60 && now.Unix()-int64(age) > 0; age++ {
		second := now.Unix() - int64(age)
		sample := r.samples[second%60]
		if sample.second != second || sample.rate > 0 {
			break
		}
		out.OutageSeconds++
	}
	return out
}

// DeliveryWindow responds to variable positive delivery and an observed outage
// only when reads are actually blocked. Idle/full buffers do not turn zero rate
// into an outage. All changes still obey the caller's existing hard budgets.
func DeliveryWindow(playbackRate, waitP95Ms float64, delivery DeliveryStats, blocked, full bool, targetSeconds, maxSeconds, safetyPct int, pieceLength int64, maxAhead int) (int, int) {
	target := targetSeconds
	if !full && delivery.Samples >= 5 && delivery.Variation > .5 {
		target = max(target, 60)
	}
	if !full && blocked && delivery.OutageSeconds >= 2 {
		target = max(target, min(maxSeconds, 90+delivery.OutageSeconds))
	}
	target = min(target, max(maxSeconds, targetSeconds))
	if full {
		delivery.Mean = 0
		waitP95Ms = 0
	}
	return AdaptiveWindow(playbackRate, delivery.Mean, waitP95Ms, target, maxSeconds, safetyPct, pieceLength, maxAhead)
}

// ConsumptionTracker samples forward progress over wall-clock time. Short
// Range bursts are accumulated; large jumps and idle gaps are excluded so a
// seek or a resume does not masquerade as sustained playback consumption.
type ConsumptionTracker struct {
	anchor     int64
	anchorTime time.Time
	rate       float64
	samples    int
}

func (t *ConsumptionTracker) Reset() { *t = ConsumptionTracker{} }

func (t *ConsumptionTracker) Observe(offset int64, now time.Time, baseline float64) {
	if offset < 0 || now.IsZero() {
		return
	}
	if t.anchorTime.IsZero() {
		t.anchor, t.anchorTime = offset, now
		return
	}
	dt := now.Sub(t.anchorTime).Seconds()
	if dt <= 0 {
		return
	}
	if offset < t.anchor {
		// Parallel older requests can finish behind the current playhead.
		if t.anchor-offset > 16*MiB {
			t.anchor, t.anchorTime = offset, now
		}
		return
	}
	delta := offset - t.anchor
	maxAdvance := float64(16 * MiB)
	if baseline > 0 && 4*baseline*dt > maxAdvance {
		maxAdvance = 4 * baseline * dt
	}
	if dt > 30 || float64(delta) > maxAdvance {
		t.anchor, t.anchorTime = offset, now
		return
	}
	if dt < 2 || delta == 0 {
		return
	}
	sample := float64(delta) / dt
	if baseline > 0 {
		// HTTP can fetch well ahead of the decoder. Keep observations useful
		// without treating a burst as a new 10x media bitrate.
		sample = math.Max(baseline/2, math.Min(sample, baseline*2))
	}
	if t.samples == 0 {
		t.rate = sample
	} else {
		t.rate = t.rate*0.75 + sample*0.25
	}
	t.samples++
	t.anchor, t.anchorTime = offset, now
}

func (t *ConsumptionTracker) Rate() (float64, string) {
	if t.samples < 3 {
		return 0, "low"
	}
	return t.rate, "stable"
}

func (t *ConsumptionTracker) RateAt(now time.Time) (float64, string) {
	if age := now.Sub(t.anchorTime); age < 0 || age > 30*time.Second {
		return 0, "low"
	}
	return t.Rate()
}

// AdaptiveWindow chooses a bounded forward window inside the existing cache.
// A zero download rate is unknown (for example an already full buffer), not a
// weak-swarm signal. The caller provides the cache-dependent maximum.
func AdaptiveWindow(playbackRate, downloadRate, waitP95Ms float64, targetSeconds, maxSeconds, safetyPct int, pieceLength int64, maxAhead int) (seconds, pieces int) {
	if playbackRate <= 0 || math.IsNaN(playbackRate) || math.IsInf(playbackRate, 0) || pieceLength <= 0 || maxAhead <= 0 {
		return 0, 0
	}
	if targetSeconds < 10 {
		targetSeconds = 45
	}
	if maxSeconds < targetSeconds {
		maxSeconds = targetSeconds
	}
	seconds = targetSeconds
	if downloadRate > 0 {
		ratio := downloadRate / playbackRate
		switch {
		case ratio < 1:
			seconds = max(seconds, 90)
		case ratio < 1.5:
			seconds = max(seconds, 60)
		case ratio > 2:
			seconds = max(30, seconds-15)
		}
	}
	if waitP95Ms > 500 {
		seconds = max(seconds, 90)
	}
	seconds = min(seconds, maxSeconds)
	if safetyPct < 100 || safetyPct > 300 {
		safetyPct = 130
	}
	bytes := playbackRate * float64(seconds) * float64(safetyPct) / 100
	pieces = int(math.Ceil(bytes / float64(pieceLength)))
	pieces = min(maxAhead, max(8, pieces))
	return seconds, pieces
}

// BufferExhaustionSeconds predicts how long a contiguous buffer lasts while
// the swarm downloads more slowly than playback consumes. The second return
// value is false when the rates are unknown or the buffer is not draining.
func BufferExhaustionSeconds(bufferBytes int64, playbackRate, downloadRate float64) (float64, bool) {
	if bufferBytes < 0 || playbackRate <= 0 || downloadRate < 0 ||
		math.IsNaN(playbackRate) || math.IsInf(playbackRate, 0) ||
		math.IsNaN(downloadRate) || math.IsInf(downloadRate, 0) ||
		downloadRate >= playbackRate {
		return 0, false
	}
	return float64(bufferBytes) / (playbackRate - downloadRate), true
}
