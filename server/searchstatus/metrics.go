package searchstatus

import "sync/atomic"

var metricSources = [...]string{"rutor", "jacred", "torznab"}
var metricStates = [...]string{"ok", "empty", "unavailable", "auth_required", "timeout", "cancelled", "invalid_response", "unsupported_query", "cooldown", "other"}
var metricOutcomes [len(metricSources)][len(metricStates)]atomic.Uint64

type OutcomeMetric struct {
	Source, State string
	Count         uint64
}

func recordMetric(status Status) {
	source := -1
	for i, name := range metricSources {
		if status.Source == name {
			source = i
			break
		}
	}
	if source < 0 {
		return
	}
	state := len(metricStates) - 1
	for i, name := range metricStates {
		if status.State == name {
			state = i
			break
		}
	}
	metricOutcomes[source][state].Add(1)
}
func OutcomeMetrics() []OutcomeMetric {
	out := make([]OutcomeMetric, 0, len(metricSources)*len(metricStates))
	for i, source := range metricSources {
		for j, state := range metricStates {
			out = append(out, OutcomeMetric{source, state, metricOutcomes[i][j].Load()})
		}
	}
	return out
}
