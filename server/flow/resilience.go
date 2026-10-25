package flow

import "math"

// ResilientDemand uses sustained HTTP progress only as a bounded safety hint
// above credible media demand. It never lowers the probed average or labels an
// HTTP read rate as a decoder clock. Unqualified/stale values remain unknown.
func ResilientDemand(estimate Estimate, observed float64, confidence string) float64 {
	rate := estimate.BytesPerSecond
	if rate <= 0 || math.IsNaN(rate) || math.IsInf(rate, 0) {
		return 0
	}
	if estimate.Confidence != "medium" && estimate.Confidence != "high" {
		return rate
	}
	if confidence == "stable" && observed > rate && !math.IsInf(observed, 0) && !math.IsNaN(observed) {
		return math.Min(2*rate, observed)
	}
	return rate
}

// ResilienceTarget preserves an observed outage reserve for up to thirty
// seconds, bounded by the user's maximum and the caller's existing cache. Full,
// idle, stale or otherwise unqualified evidence cannot enlarge the target.
func ResilienceTarget(supply DeliveryEvidence, target, maximum int) int {
	if supply.Mode != "DEMAND" || supply.AgeMs < 0 || supply.AgeMs > 2500 ||
		(supply.Confidence != "medium" && supply.Confidence != "high") {
		return target
	}
	outage := max(0, min(30, supply.RecentOutageSeconds))
	return min(max(target, maximum), target+2*outage)
}

// DeficitTarget adds a recent reserve margin, never exceeding the user's time
// limit. It complements the outage margin without counting the same gap twice.
func DeficitTarget(supply DeliveryEvidence, demand float64, target, maximum int) int {
	if !qualifiedDeficit(supply) || demand <= 0 || math.IsNaN(demand) || math.IsInf(demand, 0) {
		return target
	}
	margin := math.Ceil(reserveMargin(supply, demand, min(60, max(1, target))) / demand)
	margin = math.Min(float64(max(0, maximum-target)), margin)
	return target + int(margin)
}

// ReservePlan explains a bounded startup decision. Rates describe verified
// delivery, not a decoder clock or a promise that the swarm can sustain a file.
type ReservePlan struct {
	Known                 bool     `json:"known"`
	Reason                string   `json:"reason"`
	BaseBytes             int64    `json:"base_bytes"`
	TargetBytes           int64    `json:"target_bytes"`
	MaximumBytes          int64    `json:"maximum_bytes"`
	Limited               bool     `json:"limited"`
	DemandRate            float64  `json:"demand_rate"`
	DeliveryRate          float64  `json:"delivery_rate"`
	DeficitRate           float64  `json:"deficit_rate"`
	ForecastSeconds       int      `json:"forecast_seconds"`
	AdditionalFillSeconds *float64 `json:"additional_fill_seconds,omitempty"`
}

func finitePositive(value float64) bool {
	return value > 0 && !math.IsNaN(value) && !math.IsInf(value, 0)
}

func qualifiedSupply(e DeliveryEvidence) bool {
	return e.Mode == "DEMAND" && e.AgeMs >= 0 && e.AgeMs <= 2500 && e.Samples >= 3 &&
		(e.Confidence == "medium" || e.Confidence == "high") && e.LongRate >= 0 &&
		!math.IsNaN(e.LongRate) && !math.IsInf(e.LongRate, 0)
}

func reserveMargin(supply DeliveryEvidence, demand float64, horizon int) float64 {
	historical := float64(max(0, supply.DeficitBytes))
	projected := max(0, demand-supply.LongRate) * float64(min(60, max(1, horizon)))
	return 1.3 * max(historical, projected)
}

// PlanStartupReserve retains the normal fast-start floor for healthy/unknown
// supply. A measured deficit projects only the next bounded interval, never the
// entire remaining movie. Existing user, file, cache and wait limits still win.
func PlanStartupReserve(base, maximum int64, demand float64, horizon int, supply DeliveryEvidence) ReservePlan {
	base = max(0, base)
	maximum = max(base, maximum)
	p := ReservePlan{Reason: "UNKNOWN", BaseBytes: base, TargetBytes: base, MaximumBytes: maximum}
	if !finitePositive(demand) || !qualifiedSupply(supply) {
		return p
	}
	p.Known = true
	p.DemandRate, p.DeliveryRate = demand, supply.LongRate
	p.DeficitRate = max(0, demand-supply.LongRate)
	p.ForecastSeconds = min(60, max(1, horizon))
	p.Reason = "SUFFICIENT_SUPPLY"
	if !qualifiedDeficit(supply) {
		return p
	}
	p.Reason = "RECENT_GAPS"
	if p.DeficitRate > 0 {
		p.Reason = "SUPPLY_DEFICIT"
	}
	margin := reserveMargin(supply, demand, p.ForecastSeconds)
	spare := maximum - base
	p.Limited = margin > float64(spare)
	// Clamp before float-to-int conversion, including MaxInt64's rounded float.
	if margin >= float64(spare) {
		p.TargetBytes = maximum
	} else {
		p.TargetBytes += int64(math.Ceil(margin))
	}
	if supply.LongRate > 0 {
		seconds := float64(p.TargetBytes-base) / supply.LongRate
		if !math.IsInf(seconds, 0) && !math.IsNaN(seconds) {
			p.AdditionalFillSeconds = &seconds
		}
	}
	return p
}

func qualifiedDeficit(e DeliveryEvidence) bool {
	return qualifiedSupply(e) && e.DeficitSamples >= 3 && e.DeficitBytes > 0
}

// DeficitStartupTarget is a bounded extension of the existing startup target.
// A fast or unqualified bootstrap retains its normal latency and byte floor.
func DeficitStartupTarget(base, maximum int64, supply DeliveryEvidence) int64 {
	if base >= maximum || !qualifiedDeficit(supply) {
		return base
	}
	margin := math.Min(float64(maximum-base), math.Ceil(float64(supply.DeficitBytes)*1.3))
	return base + int64(margin)
}
