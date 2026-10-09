package flow

import (
	"bufio"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ResponseRecorder observes an HTTP response without altering Range handling.
// The underlying server remains responsible for status and body semantics.
type ResponseRecorder struct {
	http.ResponseWriter
	start       time.Time
	Status      int
	Bytes       int64
	TTFB        time.Duration
	OnFirstByte func(time.Duration)
	// OnBodyStart receives the actual single-response media start before a
	// potentially blocking body read. Errors and multipart framing are excluded.
	OnBodyStart func(int64)
	// OnProgress receives the file offset after successful body writes. It is
	// deliberately unavailable for errors and multipart framing.
	OnProgress func(int64)
}

func NewResponseRecorder(w http.ResponseWriter) *ResponseRecorder {
	return &ResponseRecorder{ResponseWriter: w, start: time.Now()}
}

func (w *ResponseRecorder) WriteHeader(status int) {
	if w.Status == 0 {
		w.Status = status
		if w.OnBodyStart != nil {
			if offset, ok := w.mediaOffset(); ok {
				w.OnBodyStart(offset)
			}
		}
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *ResponseRecorder) Write(p []byte) (int, error) {
	if w.Status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	n, err := w.ResponseWriter.Write(p)
	if n > 0 && w.Bytes == 0 {
		w.TTFB = time.Since(w.start)
		if w.OnFirstByte != nil {
			w.OnFirstByte(w.TTFB)
		}
	}
	w.Bytes += int64(n)
	if n > 0 && w.OnProgress != nil {
		if offset, ok := w.mediaOffset(); ok {
			w.OnProgress(offset)
		}
	}
	return n, err
}

func (w *ResponseRecorder) mediaOffset() (int64, bool) {
	if w.Status == http.StatusOK {
		return w.Bytes, true
	}
	if w.Status != http.StatusPartialContent {
		return 0, false
	}
	// Read the actual response, not the request hint: If-Range can cause a
	// complete 200 response, and multipart bodies contain non-media bytes.
	h := w.Header().Get("Content-Range")
	if !strings.HasPrefix(h, "bytes ") {
		return 0, false
	}
	parts := strings.SplitN(strings.TrimPrefix(h, "bytes "), "/", 2)
	if len(parts) != 2 {
		return 0, false
	}
	rng := strings.SplitN(parts[0], "-", 2)
	if len(rng) != 2 {
		return 0, false
	}
	start, err := strconv.ParseInt(rng[0], 10, 64)
	end, endErr := strconv.ParseInt(rng[1], 10, 64)
	if err != nil || endErr != nil || start < 0 || end < start || w.Bytes > end-start+1 {
		return 0, false
	}
	return start + w.Bytes, true
}

func (w *ResponseRecorder) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (w *ResponseRecorder) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func (w *ResponseRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := w.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}
