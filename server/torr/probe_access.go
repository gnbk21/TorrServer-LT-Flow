package torr

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net"
	"net/http"
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
	if req == nil || req.URL == nil || probeAccessKey == "" || req.URL.Query().Get("stat") != "ffprobe" {
		return false
	}
	host, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil || !net.ParseIP(host).IsLoopback() {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(req.URL.Query().Get("probe_key")), []byte(probeAccessKey)) == 1
}
