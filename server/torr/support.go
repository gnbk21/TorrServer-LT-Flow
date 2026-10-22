package torr

import "server/flow"

type SupportTorrent struct {
	Startup         FlowStartupStatus       `json:"startup"`
	Sessions        []FlowSessionStatus     `json:"sessions"`
	TrackerStates   map[string]int          `json:"tracker_states"`
	Timeline        flow.TimelineSnapshot   `json:"timeline"`
	Integrity       flow.IntegritySnapshot  `json:"integrity"`
	TrackerFailures []SupportNetworkFailure `json:"tracker_failures"`
	MirrorFailures  []WebSeedFailure        `json:"mirror_failures"`
}

type SupportNetworkFailure struct {
	Category            string `json:"category"`
	Code                int    `json:"code"`
	Operation           int    `json:"operation"`
	ConsecutiveFailures int    `json:"consecutive_failures"`
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
		row.Integrity = tor.FlowIntegrity()
		row.MirrorFailures = tor.FlowWebSeedFailures()
		row.TrackerFailures = []SupportNetworkFailure{}
		for _, session := range tor.FlowStatus() {
			if remaining == 0 {
				truncated = true
				break
			}
			session.Group = ""
			session.SessionID = ""
			session.Traces = nil
			row.Sessions = append(row.Sessions, session)
			remaining--
		}
		for _, tracker := range tor.FlowTrackers() {
			row.TrackerStates[tracker.Status]++
			if tracker.Status == "ERROR" {
				row.TrackerFailures = append(row.TrackerFailures, SupportNetworkFailure{flow.SafeErrorCategory(tracker.ErrorCategory), tracker.ErrorCode, tracker.Operation, tracker.FailureCount})
			}
		}
		out = append(out, row)
	}
	return out, truncated
}
