package torr

import (
	"server/lt"
	"strings"
	"testing"
)

func FuzzTrackerURLRedaction(f *testing.F) {
	f.Add("https://user:password@example.test/private-passkey?token=secret")
	f.Add("udp://example.test:80/announce")
	f.Fuzz(func(t *testing.T, raw string) {
		if len(raw) > 8192 {
			return
		}
		tor := &Torrent{}
		tor.recordTrackerAlert(&lt.Alert{URL: raw, Type: "tracker_error", Error: "credential-secret"})
		for _, summary := range tor.FlowTrackers() {
			if strings.Contains(summary.Host, "@") || strings.Contains(summary.Host, "/") || strings.Contains(summary.Host, "?") || strings.Contains(summary.Error, "credential-secret") {
				t.Fatal("unredacted URL/error")
			}
		}
	})
}
