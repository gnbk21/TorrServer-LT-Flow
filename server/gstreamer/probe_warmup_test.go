//go:build gst

package gstreamer

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestProbeRetryableError(t *testing.T) {
	if probeRetryableError(nil) {
		t.Error("nil error must not be retried")
	}
	if probeRetryableError(ErrServiceClosed) || probeRetryableError(ErrBadSource) {
		t.Error("service errors must not be retried")
	}
	if probeRetryableError(fmt.Errorf("start: %w", errDiscovererUnavailable)) {
		t.Error("a missing gst-discoverer must not be retried: waiting cannot fix it")
	}
	if !probeRetryableError(ErrProbeUnavailable) {
		t.Error("an empty probe must be retried after warming the source")
	}
}

func TestWarmProbeSourceReadsHeadAndTail(t *testing.T) {
	const size = int64(8 << 20)
	var mu sync.Mutex
	var ranges []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		ranges = append(ranges, r.Header.Get("Range"))
		mu.Unlock()
		var first, last int64
		if _, err := fmt.Sscanf(r.Header.Get("Range"), "bytes=%d-%d", &first, &last); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", first, last, size))
		w.WriteHeader(http.StatusPartialContent)
		w.Write(make([]byte, last-first+1))
	}))
	defer srv.Close()

	if !warmProbeSource(context.Background(), srv.URL, size) {
		t.Fatal("warmProbeSource reported failure on a healthy source")
	}
	mu.Lock()
	defer mu.Unlock()
	want := []string{
		fmt.Sprintf("bytes=0-%d", probeWarmupHeadBytes-1),
		fmt.Sprintf("bytes=%d-%d", size-probeWarmupTailBytes, size-1),
	}
	if len(ranges) != len(want) {
		t.Fatalf("ranges = %v, want %v", ranges, want)
	}
	for i, r := range ranges {
		if r != want[i] {
			t.Errorf("range %d = %q, want %q", i, r, want[i])
		}
	}
}

func TestWarmProbeSourceFailsWithoutHead(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "no data yet", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	if warmProbeSource(context.Background(), srv.URL, 0) {
		t.Fatal("warmProbeSource must report failure when the head cannot be read")
	}
}

// A file shorter than the warm-up window is read whole, with no tail request.
func TestWarmProbeSourceSmallFile(t *testing.T) {
	const size = int64(100 << 10)
	var mu sync.Mutex
	var ranges []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		ranges = append(ranges, r.Header.Get("Range"))
		mu.Unlock()
		var first, last int64
		fmt.Sscanf(r.Header.Get("Range"), "bytes=%d-%d", &first, &last)
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", first, last, size))
		w.WriteHeader(http.StatusPartialContent)
		w.Write(make([]byte, last-first+1))
	}))
	defer srv.Close()

	if !warmProbeSource(context.Background(), srv.URL, size) {
		t.Fatal("warmProbeSource reported failure on a small file")
	}
	mu.Lock()
	defer mu.Unlock()
	if len(ranges) != 1 || ranges[0] != fmt.Sprintf("bytes=0-%d", size-1) {
		t.Fatalf("ranges = %v, want the whole file in one request", ranges)
	}
}
