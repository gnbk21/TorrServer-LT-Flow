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
	margin := math.Ceil(float64(supply.DeficitBytes) * 1.3 / demand)
	margin = math.Min(float64(max(0, maximum-target)), margin)
	return target + int(margin)
}

func qualifiedDeficit(e DeliveryEvidence) bool {
	return e.Mode == "DEMAND" && e.AgeMs >= 0 && e.AgeMs <= 2500 && e.DeficitSamples >= 3 &&
		(e.Confidence == "medium" || e.Confidence == "high") && e.DeficitBytes > 0
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
