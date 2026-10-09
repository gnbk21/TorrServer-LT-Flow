package utils

import (
	"bytes"
	"image"
	"image/jpeg"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

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
	if ok, verified := CheckImgUrl(okSrv.URL + "/poster.jpg"); !ok || !verified {
		t.Error("jpeg: want ok and verified")
	}

	htmlSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<html>not an image</html>"))
	}))
	defer htmlSrv.Close()
	if ok, _ := CheckImgUrl(htmlSrv.URL + "/index.html"); ok {
		t.Error("html: want reject")
	}

	hangSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer hangSrv.Close()
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
