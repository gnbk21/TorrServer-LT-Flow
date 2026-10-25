package utils

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"server/settings"
	"sync/atomic"
	"testing"
)

type failedImageReader struct{ err error }

func (r failedImageReader) Read([]byte) (int, error) { return 0, r.err }

func approveImageFixture(t *testing.T, endpoint string) {
	t.Helper()
	previous := settings.BTsets()
	next := settings.CloneSettings(previous)
	if next == nil {
		next = &settings.BTSets{Flow: settings.DefaultFlowSettings()}
	}
	next.TMDBSettings.ImageURL = endpoint
	settings.StoreBTsets(next)
	t.Cleanup(func() { settings.StoreBTsets(previous) })
}

func TestImageReadTimeoutBeforeContextDeadline(t *testing.T) {
	ctx := context.Background()
	clientTimeout := fmt.Errorf("response body: %w", &net.OpError{Op: "read", Net: "tcp", Err: os.ErrDeadlineExceeded})
	for _, err := range []error{clientTimeout, fmt.Errorf("body: %w", context.DeadlineExceeded), context.Canceled} {
		if ctx.Err() != nil || !uncertainImageRead(ctx, err) {
			t.Fatalf("temporary read error must remain uncertain before context cancellation: %v", err)
		}
	}
	if uncertainImageRead(ctx, io.ErrUnexpectedEOF) {
		t.Fatal("malformed image must still be rejected")
	}
	readStatus := &imageReadStatus{Reader: failedImageReader{err: clientTimeout}}
	if _, _, err := image.DecodeConfig(readStatus); !errors.Is(err, image.ErrFormat) ||
		!uncertainImageRead(ctx, readStatus.err) || ctx.Err() != nil {
		t.Fatal("image sniffing must not erase transport timeout classification")
	}
	malformed := &imageReadStatus{Reader: bytes.NewReader([]byte("not an image"))}
	if _, _, err := image.DecodeConfig(malformed); !errors.Is(err, image.ErrFormat) || uncertainImageRead(ctx, malformed.err) {
		t.Fatal("ordinary invalid image must remain rejected")
	}
}

func TestPosterValidationDoesNotBypassUnavailableInterface(t *testing.T) {
	previous := settings.BTsets()
	flow := settings.DefaultFlowSettings()
	flow.TorrentInterface, flow.RequireTorrentInterface = "Flow-missing-poster-interface-418", true
	settings.StoreBTsets(&settings.BTSets{Flow: flow})
	t.Cleanup(func() { settings.StoreBTsets(previous) })
	var called atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called.Store(true) }))
	defer srv.Close()
	approveImageFixture(t, srv.URL)
	if ok, verified := CheckImgUrl(srv.URL); !ok || verified {
		t.Fatal("unavailable interface should retain uncertain artwork")
	}
	if called.Load() {
		t.Fatal("poster bypassed the selected interface")
	}
}

func TestCheckImgUrl(t *testing.T) {
	for _, link := range []string{
		"",
		"javascript:alert(1)",
		"data:text/html,<script>alert(1)</script>",
		"file:///etc/passwd",
	} {
		if ok, _ := CheckImgUrl(link); ok {
			t.Errorf("%q: want reject", link)
		}
	}

	var jpegBuf bytes.Buffer
	if err := jpeg.Encode(&jpegBuf, image.NewRGBA(image.Rect(0, 0, 1, 1)), nil); err != nil {
		t.Fatal(err)
	}

	okSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		w.Write(jpegBuf.Bytes())
	}))
	defer okSrv.Close()
	approveImageFixture(t, okSrv.URL)
	if ok, verified := CheckImgUrl(okSrv.URL + "/poster.jpg"); !ok || !verified {
		t.Error("jpeg: want ok and verified")
	}

	htmlSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<html>not an image</html>"))
	}))
	defer htmlSrv.Close()
	approveImageFixture(t, htmlSrv.URL)
	if ok, _ := CheckImgUrl(htmlSrv.URL + "/index.html"); ok {
		t.Error("html: want reject")
	}

	hangSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer hangSrv.Close()
	approveImageFixture(t, hangSrv.URL)
	if ok, verified := CheckImgUrl(hangSrv.URL + "/poster.jpg"); !ok || verified {
		t.Error("timeout: want ok, not verified")
	}

	slowSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		w.WriteHeader(http.StatusOK)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		<-r.Context().Done()
	}))
	defer slowSrv.Close()
	approveImageFixture(t, slowSrv.URL)
	if ok, verified := CheckImgUrl(slowSrv.URL + "/poster.jpg"); !ok || verified {
		t.Error("decode after deadline: want ok, not verified")
	}
}

func TestPosterReplacementAndTemporaryFailures(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.UserAgent() != "Mozilla/5.0" {
			t.Error("missing compatibility User-Agent")
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	approveImageFixture(t, srv.URL)
	if got := SelectPoster(srv.URL, srv.URL); got != srv.URL || calls.Load() != 0 {
		t.Fatal("unchanged poster checked")
	}
	if got := SelectPoster(srv.URL, "existing"); got != "existing" {
		t.Fatal("temporary failure erased existing poster")
	}
	if got := SelectPoster(srv.URL, ""); got != srv.URL {
		t.Fatal("uncertain new URL rejected")
	}
	if got := SelectPoster("file:///invalid", "existing"); got != "existing" {
		t.Fatal("invalid replacement erased poster")
	}
	if got := SelectPoster("", "existing"); got != "" {
		t.Fatal("explicit clear ignored")
	}
}
