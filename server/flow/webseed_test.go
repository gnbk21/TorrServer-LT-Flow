package flow

import "testing"

func TestWebSeedURLBoundaryAndIdentity(t *testing.T) {
	for _, tc := range []struct {
		url          string
		local, valid bool
	}{{"https://example.org/file?signature=secret", false, true}, {"http://127.0.0.1/file", true, true}, {"http://192.168.1.4:8080/media/", true, true}, {"http://127.0.0.1/file", false, false}, {"http://169.254.169.254/latest/meta-data/", true, false}, {"http://localhost/file", true, false}, {"http://192.168.1.4/file?command=run", true, false}, {"https://user:password@example.org/file", false, false}, {"file:///etc/passwd", false, false}, {"https://example.org/file\r\nHeader:value", false, false}} {
		_, err := ValidateWebSeed(tc.url, tc.local)
		if (err == nil) != tc.valid {
			t.Fatalf("%s: %v", tc.url, err)
		}
	}
	if id := WebSeedID("https://example.org/file?signature=secret"); len(id) != 32 || id == WebSeedID("https://example.org/other") {
		t.Fatal("invalid opaque source identity")
	}
}
