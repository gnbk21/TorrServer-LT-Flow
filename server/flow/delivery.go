package flow

import (
	"math"
	"time"
)

// DeliveryMeter counts newly verified forward bytes, not wire traffic. The
// caller serializes access and classifies actual demand before adding bytes.
// Full buffers, probes, preparation and idle time do not lower the supply rate.
type DeliveryMeter struct {
	slots [60]struct {
		second  int64
		bytes   int64
		rate    float64
		demand  bool
		tainted bool
	}
	mode     string
	observed time.Time
}

type DeliveryEvidence struct {
	Mode                string  `json:"mode"`
	ShortRate           float64 `json:"short_rate"`
	LongRate            float64 `json:"long_rate"`
	ShortSamples        int     `json:"short_samples"`
	Samples             int     `json:"samples"`
	AgeMs               int64   `json:"age_ms"`
	Confidence          string  `json:"confidence"`
	Variation           float64 `json:"variation"`
	OutageSeconds       int     `json:"outage_seconds"`
	RecentOutageSeconds int     `json:"recent_outage_seconds"`
	DeficitBytes        int64   `json:"deficit_bytes"`
	DeficitSamples      int     `json:"deficit_samples"`
}

func (m *DeliveryMeter) Reset() { *m = DeliveryMeter{} }

func (m *DeliveryMeter) Observe(now time.Time, mode string) {
	if now.Unix() <= 0 {
		return
	}
	m.mode, m.observed = mode, now
	s := &m.slots[now.Unix()%60]
	if s.second != now.Unix() {
		s.second, s.bytes, s.demand, s.tainted = now.Unix(), 0, false, false
		s.rate = 0
	}
	// An interval that included non-demand time is not a clean supply sample.
	if mode == "DEMAND" {
		s.demand = !s.tainted
	} else {
		s.demand = false
		s.tainted = true
		s.bytes = 0
	}
}

// ObserveDemand records the demand applicable to this interval, rather than
// retroactively applying today's rate to yesterday's deliveries. Missing or
// unqualified rates cannot manufacture a reserve estimate.
func (m *DeliveryMeter) ObserveDemand(now time.Time, mode string, rate float64) {
	m.Observe(now, mode)
	if now.Unix() <= 0 || mode != "DEMAND" || rate <= 0 || math.IsNaN(rate) || math.IsInf(rate, 0) {
		return
	}
	s := &m.slots[now.Unix()%60]
	if s.demand && !s.tainted {
		s.rate = max(s.rate, rate)
	}
}

func (m *DeliveryMeter) AddVerified(bytes int64, now time.Time) {
	if bytes <= 0 || m.mode != "DEMAND" || now.Sub(m.observed) < 0 || now.Sub(m.observed) > 2*time.Second {
		return
	}
	s := &m.slots[now.Unix()%60]
	if s.second != now.Unix() {
		s.second, s.bytes, s.demand, s.tainted = now.Unix(), 0, true, false
		s.rate = 0
	}
	if s.demand && bytes <= math.MaxInt64-s.bytes {
		s.bytes += bytes
	}
}

func (m *DeliveryMeter) Snapshot(now time.Time) DeliveryEvidence {
	out := DeliveryEvidence{Mode: m.mode, Confidence: "unknown", AgeMs: -1}
	if out.Mode == "" {
		out.Mode = "IDLE"
	}
	if !m.observed.IsZero() {
		out.AgeMs = now.Sub(m.observed).Milliseconds()
	}
	var sum, squares float64
	positive := 0
	for age := int64(1); age < 60; age++ {
		second := now.Unix() - age
		if second <= 0 {
			break
		}
		s := m.slots[second%60]
		if s.second != second || !s.demand {
			continue
		}
		out.Samples++
		out.LongRate += float64(s.bytes)
		if age <= 5 {
			out.ShortSamples++
			out.ShortRate += float64(s.bytes)
		}
		if s.bytes > 0 {
			positive++
			sum += float64(s.bytes)
			squares += float64(s.bytes) * float64(s.bytes)
		}
	}
	if out.Samples > 0 {
		out.LongRate /= float64(out.Samples)
	}
	if out.ShortSamples > 0 {
		out.ShortRate /= float64(out.ShortSamples)
	}
	if positive >= 5 {
		mean := sum / float64(positive)
		out.Variation = math.Sqrt(math.Max(0, squares/float64(positive)-mean*mean)) / mean
	}
	for age := int64(1); age < 60; age++ {
		second := now.Unix() - age
		if second <= 0 {
			break
		}
		s := m.slots[second%60]
		if s.second != second || !s.demand || s.bytes > 0 {
			break
		}
		out.OutageSeconds++
	}
	// Remember recent consecutive delivery gaps after supply returns. Otherwise
	// a reserve can contract immediately after the very outage it should absorb.
	run := 0
	for age := int64(1); age <= 30 && now.Unix()-age > 0; age++ {
		second := now.Unix() - age
		s := m.slots[second%60]
		if s.second == second && s.demand && !s.tainted && s.bytes == 0 {
			run++
			out.RecentOutageSeconds = max(out.RecentOutageSeconds, run)
		} else {
			run = 0
		}
	}
	if out.Mode == "DEMAND" && out.AgeMs >= 0 && out.AgeMs <= 2500 && out.ShortSamples >= 3 {
		out.Confidence = "medium"
		if out.ShortSamples == 5 && out.Samples >= 10 {
			out.Confidence = "high"
		}
	}
	// Maximum cumulative drawdown in the recent thirty-second horizon. Surplus
	// replenishes reserve; gaps/non-demand intervals split runs instead of
	// bridging an idle, probe, seek or unobserved interval with fictitious demand.
	var deficit, peak float64
	for age := int64(30); age >= 1; age-- {
		second := now.Unix() - age
		if second <= 0 {
			deficit = 0
			continue
		}
		s := m.slots[second%60]
		if s.second != second || !s.demand || s.tainted || s.rate <= 0 {
			deficit = 0
			continue
		}
		out.DeficitSamples++
		deficit = max(0, deficit+s.rate-float64(s.bytes))
		peak = max(peak, deficit)
	}
	// float64(MaxInt64) rounds upward, so clamp before converting.
	if peak >= float64(math.MaxInt64) {
		out.DeficitBytes = math.MaxInt64
	} else {
		out.DeficitBytes = int64(math.Ceil(peak))
	}
	return out
}

// RiskDecision is explainable and uses qualified supply only. HTTP delivery is
// an observation, never an authenticated decoder clock.
type RiskDecision struct {
	Level             string   `json:"level"`
	Reason            string   `json:"reason"`
	Score             int      `json:"score"`
	Confidence        string   `json:"confidence"`
	TargetSeconds     int      `json:"target_seconds"`
	ExhaustionSeconds *float64 `json:"exhaustion_seconds,omitempty"`
}

func BufferRisk(bufferBytes int64, demand, waitMs float64, suppliers *int, supply DeliveryEvidence, target, maximum int) RiskDecision {
	r := RiskDecision{Level: "UNKNOWN", Reason: "INSUFFICIENT_EVIDENCE", Confidence: supply.Confidence, TargetSeconds: target}
	if demand <= 0 || math.IsNaN(demand) || math.IsInf(demand, 0) {
		r.Confidence = "unknown"
		return r
	}
	if supply.Confidence != "medium" && supply.Confidence != "high" {
		r.Confidence = "unknown"
		return r
	}
	rate := supply.LongRate
	if supply.ShortSamples >= 3 {
		rate = math.Min(rate, supply.ShortRate)
	}
	r.Level, r.Reason = "HEALTHY", "SUFFICIENT_SUPPLY"
	if rate < demand {
		r.Score += 40
		r.Reason = "SUPPLY_DEFICIT"
	}
	if float64(bufferBytes)/demand < 15 {
		r.Score += 25
	}
	if waitMs > 500 {
		r.Score += 15
		r.Reason = "PIECE_WAIT"
	}
	if supply.Variation > .5 {
		r.Score += 15
	}
	if suppliers != nil && *suppliers <= 1 {
		r.Score += 10
	}
	if seconds, ok := BufferExhaustionSeconds(bufferBytes, demand, rate); ok {
		r.ExhaustionSeconds = &seconds
	}
	if r.Score >= 60 {
		r.Level = "HIGH"
		r.TargetSeconds = max(target, 120)
	} else if r.Score >= 30 {
		r.Level = "ELEVATED"
		r.TargetSeconds = max(target, 90)
	}
	if r.ExhaustionSeconds != nil && *r.ExhaustionSeconds < 30 {
		r.Level = "HIGH"
		r.Reason = "BUFFER_DRAINING"
	}
	r.Score = min(100, r.Score)
	r.TargetSeconds = min(max(target, maximum), r.TargetSeconds)
	return r
}
