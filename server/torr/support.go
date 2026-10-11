package torr

import "server/flow"

type SupportTorrent struct {
	Startup       FlowStartupStatus     `json:"startup"`
	Sessions      []FlowSessionStatus   `json:"sessions"`
	TrackerStates map[string]int        `json:"tracker_states"`
	Timeline      flow.TimelineSnapshot `json:"timeline"`
}

// Use live instances only. Do not activate saved torrents to generate a report.
// Names, hashes, URLs, tokens, client groups and request traces are excluded.
func SupportPlayback() ([]SupportTorrent, bool) {
	out := []SupportTorrent{}
	truncated := false
	remaining := 256
	if helperEngine() == nil {
		return out, false
	}
	for _, tor := range helperEngine().ListTorrents() {
		if tor == nil {
			continue
		}
		if len(out) >= 64 {
			truncated = true
			break
		}
		row := SupportTorrent{Startup: tor.FlowStartup(), Sessions: []FlowSessionStatus{}, TrackerStates: map[string]int{}, Timeline: tor.FlowTimeline()}
		for _, session := range tor.FlowStatus() {
			if remaining == 0 {
				truncated = true
				break
			}
			session.Group = ""
			session.Traces = nil
			row.Sessions = append(row.Sessions, session)
			remaining--
		}
		for _, tracker := range tor.FlowTrackers() {
			row.TrackerStates[tracker.Status]++
		}
		out = append(out, row)
	}
	return out, truncated
}
