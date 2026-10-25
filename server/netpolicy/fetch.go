package netpolicy

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// FetchClient grants LAN access only to an explicitly selected literal LAN
// address (or localhost), at the original scheme/host/port. Public hostnames
// cannot rebind to LAN, and redirects never inherit a different origin's grant.
// Call CloseIdleConnections when the operation finishes. Streaming peer traffic
// is not routed through this client.
func FetchClient(endpoint string, configured bool, timeout time.Duration) (*http.Client, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, errors.New("invalid HTTP destination")
	}
	local := configured && (approvedLiteralLAN(u.Hostname(), true) || strings.EqualFold(u.Hostname(), "localhost"))
	if err := ValidateDestination(u, local); err != nil {
		return nil, err
	}
	t := newDestinationTransport(false, u, local)
	return &http.Client{Timeout: timeout, Transport: t, CheckRedirect: func(next *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return errors.New("HTTP redirect limit reached")
		}
		if u.Scheme == "https" && next.URL.Scheme != "https" {
			return errors.New("HTTPS downgrade redirect refused")
		}
		return ValidateDestination(next.URL, local && sameOrigin(u, next.URL))
	}}, nil
}

func sameOrigin(a, b *url.URL) bool {
	return strings.EqualFold(a.Scheme, b.Scheme) && strings.EqualFold(a.Host, b.Host)
}
