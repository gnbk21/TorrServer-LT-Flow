package torr

import (
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"sort"
	"strings"
	"time"

	"server/lt"
)

type FlowTrackerDiagnostic struct {
	ID       string    `json:"id"`
	Protocol string    `json:"protocol"`
	Host     string    `json:"host"`
	Status   string    `json:"status"`
	Peers    int       `json:"peers"`
	Error    string    `json:"error,omitempty"`
	LastAt   time.Time `json:"last_at"`
}

// recordTrackerAlert retains only a bounded, credential-safe summary. Tracker
// passkeys commonly occur in URL paths or queries and must not reach the API.
func (t *Torrent) recordTrackerAlert(a *lt.Alert) {
	if t == nil || a == nil || a.URL == "" {
		return
	}
	parsed, err := url.Parse(a.URL)
	if err != nil || parsed.Scheme == "" {
		return
	}
	digest := sha256.Sum256([]byte(a.URL))
	id := hex.EncodeToString(digest[:6])
	d := FlowTrackerDiagnostic{ID: id, Protocol: strings.ToLower(parsed.Scheme),
		Host: parsed.Hostname(), LastAt: time.Now()}
	if d.Host == "" {
		d.Host = "unknown"
	}
	switch a.Type {
	case "tracker_reply", "tracker_reply_alert":
		d.Status, d.Peers = "OK", a.Peers
	case "tracker_error", "tracker_error_alert":
		d.Status, d.Error = "ERROR", safeTrackerError(a.Error)
	default:
		return
	}
	t.trackerMu.Lock()
	if t.trackers == nil {
		t.trackers = make(map[string]FlowTrackerDiagnostic)
	}
	if _, exists := t.trackers[id]; !exists && len(t.trackers) >= 64 {
		var oldest string
		var oldestAt time.Time
		for key, prior := range t.trackers {
			if oldest == "" || prior.LastAt.Before(oldestAt) {
				oldest, oldestAt = key, prior.LastAt
			}
		}
		delete(t.trackers, oldest)
	}
	t.trackers[id] = d
	t.trackerMu.Unlock()
}

func safeTrackerError(message string) string {
	message = strings.ToLower(message)
	switch {
	case strings.Contains(message, "resolve"), strings.Contains(message, "dns"):
		return "DNS"
	case strings.Contains(message, "timed out"), strings.Contains(message, "timeout"):
		return "TIMEOUT"
	case strings.Contains(message, "certificate"), strings.Contains(message, "tls"), strings.Contains(message, "ssl"):
		return "TLS"
	case strings.Contains(message, "refused"), strings.Contains(message, "connect"):
		return "CONNECTION"
	default:
		return "TRACKER_ERROR"
	}
}

func (t *Torrent) FlowTrackers() []FlowTrackerDiagnostic {
	if t == nil {
		return nil
	}
	t.trackerMu.Lock()
	out := make([]FlowTrackerDiagnostic, 0, len(t.trackers))
	for _, d := range t.trackers {
		out = append(out, d)
	}
	t.trackerMu.Unlock()
	sort.Slice(out, func(i, j int) bool { return out[i].LastAt.After(out[j].LastAt) })
	return out
}
