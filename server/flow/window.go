package flow

import (
	"math"
	"time"
)

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
