package auth

import (
	"io"
	"net/http"
	"time"
)

// Go's HTTP/1 server can close a Connection: close request without consuming
// its body. On Windows, unread TCP bytes can then reset the connection before
// the client receives a rejection. Consume only small, fixed-length bodies, with a
// deadline; never wait for an upload or solicit an Expect: 100-continue body.
func discardRejectedBody(w http.ResponseWriter, r *http.Request) {
	const limit = 4096
	if !r.Close || r.Body == nil || r.ContentLength <= 0 || r.ContentLength > limit || len(r.TransferEncoding) != 0 || r.Header.Get("Expect") != "" {
		return
	}
	controller := http.NewResponseController(w)
	if controller.SetReadDeadline(time.Now().Add(250*time.Millisecond)) != nil {
		return // A wrapper without deadline support must not introduce a wait.
	}
	defer controller.SetReadDeadline(time.Time{})
	_, _ = io.CopyN(io.Discard, r.Body, r.ContentLength)
}
