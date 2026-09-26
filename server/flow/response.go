package flow

import (
	"bufio"
	"net"
	"net/http"
	"time"
)

// ResponseRecorder observes an HTTP response without altering Range handling.
// The underlying server remains responsible for status and body semantics.
type ResponseRecorder struct {
	http.ResponseWriter
	start  time.Time
	Status int
	Bytes  int64
	TTFB   time.Duration
}

func NewResponseRecorder(w http.ResponseWriter) *ResponseRecorder {
	return &ResponseRecorder{ResponseWriter: w, start: time.Now()}
}

func (w *ResponseRecorder) WriteHeader(status int) {
	if w.Status == 0 {
		w.Status, w.TTFB = status, time.Since(w.start)
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *ResponseRecorder) Write(p []byte) (int, error) {
	if w.Status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	n, err := w.ResponseWriter.Write(p)
	w.Bytes += int64(n)
	return n, err
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
