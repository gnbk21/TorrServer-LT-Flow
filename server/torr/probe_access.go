package torr

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net"
	"net/http"
	"net/url"
)

var probeAccessKey = func() string {
	var key [32]byte
	if _, err := rand.Read(key[:]); err != nil {
		return ""
	}
	return hex.EncodeToString(key[:])
}()

// Only a process-secret loopback request can bypass playback policy or receive
// the internal probe reader group. A public query marker is insufficient.
func IsInternalProbe(req *http.Request) bool {
	if req == nil || req.URL == nil || probeAccessKey == "" {
		return false
	}
	kind := req.URL.Query().Get("stat")
	if kind != "ffprobe" && kind != "gstreamer" {
		return false
	}
	host, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil || !net.ParseIP(host).IsLoopback() {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(req.URL.Query().Get("probe_key")), []byte(probeAccessKey)) == 1
}

// InternalMediaURL grants this process access to its loopback media endpoint.
// It must never be returned in playlists, diagnostics or public API responses.
func InternalMediaURL(base string, probe bool) string {
	u, err := url.Parse(base)
	if err != nil {
		return ""
	}
	q := u.Query()
	kind := "gstreamer"
	if probe {
		kind = "ffprobe"
	}
	q.Set("stat", kind)
	q.Set("probe_key", probeAccessKey)
	u.RawQuery = q.Encode()
	return u.String()
}
