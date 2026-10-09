package settings

import "sync/atomic"

func HTTPEnabled() bool { return !Ssl || Args == nil || !Args.HTTPSOnly }
func PlainHTTPServesMedia() bool {
	return HTTPEnabled() && (!Ssl || Args == nil || !Args.ForceHTTPS || Args.HTTPMedia)
}

var internalBaseURL atomic.Pointer[string]

// SetInternalBaseURL is used by the listener lifecycle, not by settings drafts.
func SetInternalBaseURL(base string) { internalBaseURL.Store(&base) }
func LoopbackBaseURL() string {
	if base := internalBaseURL.Load(); base != nil && *base != "" {
		return *base
	}
	if PlainHTTPServesMedia() {
		return "http://127.0.0.1:" + Port
	}
	return "https://127.0.0.1:" + SslPort
}
