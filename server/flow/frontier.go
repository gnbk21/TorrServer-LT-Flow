package flow

import (
	"math"
	"time"
)

// FrontierGrowth measures advancement of a contiguous byte frontier. It is
// independent of HTTP throughput and cannot claim a decoder consumption rate.
type FrontierGrowth struct {
	end int64
	at  time.Time
}

func (f *FrontierGrowth) Reset() { *f = FrontierGrowth{} }
func (f *FrontierGrowth) Observe(start, end int64, demand bool, now time.Time) *float64 {
	previous, at := f.end, f.at
	f.end, f.at = end, now
	if !demand || at.IsZero() || start > previous || end < previous || now.Sub(at) < 500*time.Millisecond || now.Sub(at) > 5*time.Second {
		if !demand {
			f.Reset()
		}
		return nil
	}
	rate := float64(end-previous) / now.Sub(at).Seconds()
	return &rate
}

// UrgentHorizon retains an ordinary forward pipeline. Queue time is a predicted
// service horizon, not RTT. Missing/stale evidence is handled by the caller;
// invalid demand retains the baseline. Never collapse to one urgent piece.
func UrgentHorizon(rate float64, pieceLength int64, forward int, queueMs int64) int {
	if rate <= 0 || math.IsNaN(rate) || math.IsInf(rate, 0) || pieceLength <= 0 || queueMs < 0 {
		return forward
	}
	seconds := float64(max(int64(2000), min(int64(9000), queueMs)+1000)) / 1000
	pieces := rate*seconds/float64(pieceLength) + 2
	if pieces >= float64(forward) {
		return forward
	}
	return min(forward, max(2, int(pieces+0.999999)))
}
