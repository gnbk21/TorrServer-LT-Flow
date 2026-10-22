package torr

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"server/lt"
)

func TestTrackerCategoriesRemainDistinctAndSafe(t *testing.T) {
	tor := &Torrent{}
	for _, category := range []string{"HTTP", "DNS", "ASIO", "LIBTORRENT", "secret.invalid/token"} {
		tor.recordTrackerAlert(&lt.Alert{Type: "tracker_error", URL: "https://tracker.example/a", ErrorCategory: category, ErrorCode: 503, FailureCount: 3})
		got := tor.FlowTrackers()[0]
		if got.ErrorCode != 503 || got.FailureCount != 3 {
			t.Fatal(got)
		}
		if category == "HTTP" && got.Error != "HTTP_5XX" {
			t.Fatal(got)
		}
		if category != "HTTP" && got.Error == "HTTP_5XX" {
			t.Fatal("code interpreted without category")
		}
		if category == "secret.invalid/token" && got.ErrorCategory != "UNKNOWN" {
			t.Fatal("unbounded native category")
		}
	}
}

func TestTrackerDiagnosticsRedactCredentialsAndBoundHistory(t *testing.T) {
	tor := &Torrent{}
	secretURL := "https://user:password@tracker.example/announce/secret-key?passkey=another-secret"
	tor.recordTrackerAlert(&lt.Alert{Type: "tracker_reply", URL: secretURL, Peers: 17})
	items := tor.FlowTrackers()
	if len(items) != 1 || items[0].Protocol != "https" || items[0].Host != "tracker.example" || items[0].Peers != 17 {
		t.Fatalf("tracker reply summary: %+v", items)
	}
	tor.recordTrackerAlert(&lt.Alert{Type: "tracker_error", URL: secretURL, Error: "failed at " + secretURL})
	encoded, err := json.Marshal(tor.FlowTrackers())
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"password", "secret-key", "another-secret"} {
		if strings.Contains(string(encoded), secret) {
			t.Fatalf("tracker diagnostic disclosed %q", secret)
		}
	}
	for i := 0; i < 70; i++ {
		tor.recordTrackerAlert(&lt.Alert{Type: "tracker_reply", URL: fmt.Sprintf("udp://tracker%d.example:80/announce", i)})
	}
	if got := len(tor.FlowTrackers()); got != 64 {
		t.Fatalf("history size = %d", got)
	}
}
