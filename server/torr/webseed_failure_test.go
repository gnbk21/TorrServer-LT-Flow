package torr

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"server/lt"
)

func TestMirrorFailureEvidenceIsBoundedAndHasNoURLs(t *testing.T) {
	tor := &Torrent{}
	for i := 0; i < 20; i++ {
		tor.recordWebSeedFailure(&lt.Alert{URL: fmt.Sprintf("https://mirror.example/private/path/%d?token=secret", i), ErrorCategory: "HTTP", ErrorCode: 403})
	}
	if len(tor.FlowWebSeedFailures()) != 16 {
		t.Fatal("unbounded mirror history")
	}
	encoded, err := json.Marshal(tor.FlowWebSeedFailures())
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"mirror.example", "private", "token", "secret"} {
		if strings.Contains(string(encoded), secret) {
			t.Fatal("source identity leaked")
		}
	}
	if tor.FlowWebSeedFailures()[0].Category != "HTTP" || tor.FlowWebSeedFailures()[0].Code != 403 {
		t.Fatal("lost error category/code")
	}
}
