package log

import (
	"strings"
	"testing"
)

func TestRedactSecretsCoversPathQueryUserinfoAndCapabilities(t *testing.T) {
	for _, sample := range []string{
		"Get https://alice:secret@tracker.example/path-secret/announce?passkey=query-secret failed",
		"udp://tracker.example/secret-key/announce",
		"magnet:?xt=urn:btih:abc&tr=https%3A%2F%2Ftracker.example%2Fsecret",
		"probe_key=secret token=secret password=secret Authorization:secret",
		"/flow/play/secret",
		`{"password":"secret with spaces", "api_key": "secret"}`,
		`Authorization: Bearer secret`,
		`authorization=Basic secret`,
		`password='secret with spaces'`,
		"socks5h://alice:secret@proxy.example:1080",
		"wss://alice:secret@tracker.example/private-secret",
		"ss=secret TGToken=secret bot_token=secret session_token=secret client_token=secret",
	} {
		out := RedactSecrets(sample)
		if strings.Contains(out, "secret") || strings.Contains(out, "alice") {
			t.Fatalf("secret survived: %q", out)
		}
	}
	if got := RedactSecrets("metadata unavailable after 20s"); got != "metadata unavailable after 20s" {
		t.Fatal(got)
	}
	if got := RedactSecrets("address=192.0.2.1 progress=20"); got != "address=192.0.2.1 progress=20" {
		t.Fatal(got)
	}
}
