package web

import (
	"net"
	"net/http"
	"net/url"
	"strings"

	"server/settings"
)

// mediaPathPrefixes stay on plain HTTP with --force-https --http-media, as do the
// GStreamer HLS paths under /gst/<hash>/ (see isMediaPath).
var mediaPathPrefixes = []string{"/stream/", "/play/", "/playlist/", "/playlistall/", "/flow/play/"}

// forceHTTPSHandler redirects every request to HTTPS. With httpMedia, media paths are
// served with h instead, for players that can't use HTTPS. There are no exceptions by
// client address: behind a local reverse proxy or Docker port publishing, internet
// clients appear as loopback or private IPs.
func forceHTTPSHandler(h http.Handler, httpMedia bool) http.Handler {
	redirect := httpsRedirectHandler()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if httpMedia && (r.Method == http.MethodGet || r.Method == http.MethodHead) && isMediaRequest(r) {
			h.ServeHTTP(w, r)
			return
		}
		redirect.ServeHTTP(w, r)
	})
}

func isMediaPath(p string) bool {
	if p == "/stream" || p == "/playlist" {
		return true
	}
	// GStreamer HLS: /gst/<hash>/master.m3u8, video.m3u8, init.mp4, seg/, subs/, heartbeat.
	// Control endpoints (/gst/settings, /gst/remove, /gst/echo) have no second segment.
	if rest, ok := strings.CutPrefix(p, "/gst/"); ok {
		_, action, ok := strings.Cut(rest, "/")
		if !ok {
			return false
		}
		return action == "master.m3u8" || action == "video.m3u8" || action == "init.mp4" || strings.HasPrefix(action, "seg/") || strings.HasPrefix(action, "subs/") || action == "heartbeat"
	}
	for _, prefix := range mediaPathPrefixes {
		if strings.HasPrefix(p, prefix) {
			return true
		}
	}
	return false
}

func isMediaRequest(r *http.Request) bool {
	if r.URL.Path == "/stream" || strings.HasPrefix(r.URL.Path, "/stream/") {
		q := r.URL.Query()
		_, play := q["play"]
		_, m3u := q["m3u"]
		_, save := q["save"]
		_, preload := q["preload"]
		return (play || m3u) && !save && !preload
	}
	return isMediaPath(r.URL.Path)
}

// httpsRedirectHandler redirects every request to the same host and path on the HTTPS port.
func httpsRedirectHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, buildHTTPSRedirectTarget(r), http.StatusTemporaryRedirect)
	})
}

func buildHTTPSRedirectTarget(r *http.Request) string {
	host := r.Host
	hostName, _, err := net.SplitHostPort(host)
	if err != nil {
		hostName = strings.TrimSuffix(strings.TrimPrefix(host, "["), "]")
	}
	sslPort := settings.SslPort
	if sslPort == "" {
		sslPort = "8091"
	}
	var httpsHost string
	if sslPort == "443" {
		httpsHost = hostName
		if strings.Contains(hostName, ":") {
			httpsHost = "[" + hostName + "]"
		}
	} else {
		httpsHost = net.JoinHostPort(hostName, sslPort)
	}
	u := &url.URL{
		Scheme:   "https",
		Host:     httpsHost,
		Path:     r.URL.Path,
		RawPath:  r.URL.RawPath,
		RawQuery: r.URL.RawQuery,
	}
	if u.Path == "" {
		u.Path = "/"
	}
	return u.String()
}
