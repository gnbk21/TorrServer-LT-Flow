package mimetype

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestUntrustedStreamContentAndRanges(t *testing.T) {
	for _, tc := range []struct {
		name, contentType string
		attachment        bool
	}{
		{"movie.mkv", "video/x-matroska", false},
		{"movie.ts", "video/mpeg", false},
		{"movie.avi", "video/avi", false},
		{"track.mp3", "audio/mpeg", false},
		{"cover.png", "image/png", false},
		{"subtitle.srt", "text/plain; charset=utf-8", false},
		{"subtitle.vtt", "text/vtt; charset=utf-8", false},
		{"attack.html", "application/octet-stream", true},
		{"attack.svg", "application/octet-stream", true},
		{"no-extension", "application/octet-stream", true},
		{"..\\attack.html.part", "application/octet-stream", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := []byte("<script>alert(document.domain)</script>")
			req := httptest.NewRequest(http.MethodGet, "/stream", nil)
			req.Header.Set("Range", "bytes=2-8")
			resp := httptest.NewRecorder()
			SetStreamHeaders(resp.Header(), tc.name)
			http.ServeContent(resp, req, tc.name, time.Time{}, bytes.NewReader(body))
			if resp.Code != http.StatusPartialContent || !bytes.Equal(resp.Body.Bytes(), body[2:9]) {
				t.Fatalf("Range changed: status=%d body=%q", resp.Code, resp.Body.String())
			}
			if got := resp.Header().Get("Content-Type"); got != tc.contentType {
				t.Fatalf("Content-Type=%q want %q", got, tc.contentType)
			}
			if resp.Header().Get("X-Content-Type-Options") != "nosniff" ||
				(resp.Header().Get("Content-Disposition") == "attachment") != tc.attachment {
				t.Fatalf("unsafe download headers: %v", resp.Header())
			}
		})
	}
}
