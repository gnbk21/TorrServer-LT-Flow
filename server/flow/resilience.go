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
