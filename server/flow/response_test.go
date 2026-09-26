package flow

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"strings"
	"testing"
	"time"
)

func TestServeContentRangesWithRecorder(t *testing.T) {
	data := "abcdefghijklmnopqrstuvwxyz"
	for _, tc := range []struct {
		rng    string
		status int
		want   string
	}{
		{"bytes=0-", 206, data},
		{"bytes=3-5", 206, "def"},
		{"bytes=-3", 206, "xyz"},
		{"bytes=3-5,8-10", 206, ""},
		{"bytes=100-", 416, ""},
	} {
		r := httptest.NewRequest(http.MethodGet, "/play", nil)
		r.Header.Set("Range", tc.rng)
		out := httptest.NewRecorder()
		recorder := NewResponseRecorder(out)
		recorder.Header().Set("ETag", `"stable"`)
		recorder.Header().Set("Accept-Ranges", "bytes")
		http.ServeContent(recorder, r, "movie.mkv", time.Unix(0, 0), strings.NewReader(data))
		if recorder.Status != tc.status {
			t.Fatalf("%s: status %d", tc.rng, recorder.Status)
		}
		if tc.want != "" && out.Body.String() != tc.want {
			t.Fatalf("%s: body %q", tc.rng, out.Body.String())
		}
		if out.Header().Get("Connection") != "" {
			t.Fatalf("%s: forced close", tc.rng)
		}
		if tc.rng == "bytes=3-5" && (out.Header().Get("Content-Range") != "bytes 3-5/26" || out.Header().Get("Content-Length") != "3") {
			t.Fatalf("incorrect range headers: %v", out.Header())
		}
	}
}

func TestRecorderAllowsHTTPConnectionReuse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw := NewResponseRecorder(w)
		http.ServeContent(rw, r, "movie.mkv", time.Unix(0, 0), strings.NewReader("abcdefgh"))
	}))
	defer srv.Close()
	transport := &http.Transport{}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport}
	for i := 0; i < 2; i++ {
		req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
		req.Header.Set("Range", "bytes=0-3")
		reused := false
		req = req.WithContext(httptrace.WithClientTrace(req.Context(), &httptrace.ClientTrace{GotConn: func(info httptrace.GotConnInfo) { reused = info.Reused }}))
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.Copy(io.Discard, resp.Body); err != nil {
			t.Fatal(err)
		}
		if err := resp.Body.Close(); err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusPartialContent {
			t.Fatalf("status %d", resp.StatusCode)
		}
		if i == 1 && !reused {
			t.Fatal("second Range request did not reuse the HTTP connection")
		}
	}
}

func TestRecorderTTFBIncludesWaitAfterHeaders(t *testing.T) {
	r := NewResponseRecorder(httptest.NewRecorder())
	called := 0
	r.OnFirstByte = func(time.Duration) { called++ }
	r.WriteHeader(http.StatusPartialContent)
	time.Sleep(15 * time.Millisecond)
	if _, err := r.Write([]byte("x")); err != nil {
		t.Fatal(err)
	}
	if r.TTFB < 15*time.Millisecond {
		t.Fatalf("TTFB %v excluded the content wait", r.TTFB)
	}
	if _, err := r.Write([]byte("y")); err != nil {
		t.Fatal(err)
	}
	if called != 1 {
		t.Fatalf("first-byte callback called %d times", called)
	}
}
