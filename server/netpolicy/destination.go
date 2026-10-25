package netpolicy

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"
)

var excludedNetworks = []netip.Prefix{
	netip.MustParsePrefix("64:ff9b::/96"), netip.MustParsePrefix("64:ff9b:1::/48"),
	netip.MustParsePrefix("2001::/23"), netip.MustParsePrefix("2002::/16"),
	netip.MustParsePrefix("100.64.0.0/10"), netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("192.0.0.0/24"), netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("198.51.100.0/24"), netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("240.0.0.0/4"), netip.MustParsePrefix("2001:db8::/32"),
}

var ErrDestination = errors.New("HTTP destination is not allowed")

func publicDestination(ip netip.Addr) bool {
	ip = ip.Unmap()
	if ip.Zone() != "" || !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
		return false
	}
	for _, prefix := range excludedNetworks {
		if prefix.Contains(ip) {
			return false
		}
	}
	return true
}
func ValidateDestination(u *url.URL, allowLAN bool) error {
	if u == nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil || u.Opaque != "" || u.Fragment != "" {
		return fmt.Errorf("%w: HTTP(S) without credentials or fragments required", ErrDestination)
	}
	if ip, err := netip.ParseAddr(u.Hostname()); err == nil && !publicDestination(ip) {
		ip = ip.Unmap()
		if !allowLAN || !(ip.IsPrivate() || ip.IsLoopback()) || ip.Zone() != "" {
			return fmt.Errorf("%w: destination requires explicit LAN approval", ErrDestination)
		}
	}
	return nil
}

// DestinationTransport pins validated resolver answers when dialing. URL-only
// validation cannot prevent DNS rebinding. TLS still verifies the URL hostname.
// Environment proxies are excluded because their resolver would bypass this gate.
type destinationTransport struct {
	*http.Transport
	bound     *boundTransport
	allowLAN  bool
	origin    *url.URL
	originLAN bool
}

func approvedLiteralLAN(host string, allow bool) bool {
	ip, err := netip.ParseAddr(host)
	ip = ip.Unmap()
	return allow && err == nil && ip.Zone() == "" && (ip.IsPrivate() || ip.IsLoopback())
}

func (t *destinationTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.originLAN && strings.EqualFold(t.origin.Hostname(), req.URL.Hostname()) && !sameOrigin(t.origin, req.URL) {
		return nil, fmt.Errorf("%w: LAN permission is limited to the configured origin", ErrDestination)
	}
	local := t.originLAN && sameOrigin(t.origin, req.URL)
	if err := ValidateDestination(req.URL, t.allowLAN || local); err != nil {
		return nil, err
	}
	// An explicitly approved literal LAN endpoint remains reachable when the
	// torrent interface is absent. Hostnames and redirects get no such exemption.
	if approvedLiteralLAN(req.URL.Hostname(), t.allowLAN) {
		return t.Transport.RoundTrip(req)
	}
	return t.bound.RoundTrip(req)
}

func DestinationTransport(allowLAN bool) *destinationTransport {
	return newDestinationTransport(allowLAN, nil, false)
}

func newDestinationTransport(allowLAN bool, origin *url.URL, originLAN bool) *destinationTransport {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.Proxy = nil
	t.ResponseHeaderTimeout = 10 * time.Second
	t.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, errors.New("invalid destination")
		}
		lan := approvedLiteralLAN(host, allowLAN)
		configuredLAN := originLAN && strings.EqualFold(address, origin.Host)
		// URL.Host can omit the default port while the dial address includes it.
		if originLAN && origin.Port() == "" {
			portDefault := "80"
			if origin.Scheme == "https" {
				portDefault = "443"
			}
			configuredLAN = strings.EqualFold(host, origin.Hostname()) && port == portDefault
		}
		s := Snapshot()
		if !lan && s.Name != "" && s.State != "BOUND" {
			return nil, errors.New("selected torrent interface is unavailable")
		}
		addresses, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
		if err != nil {
			return nil, errors.New("destination resolution failed")
		}
		if len(addresses) == 0 {
			return nil, errors.New("destination has no addresses")
		}
		for _, ip := range addresses {
			ip = ip.Unmap()
			approved := configuredLAN && (ip.IsPrivate() || ip.IsLoopback()) && ip.Zone() == ""
			if !publicDestination(ip) && !lan && !approved {
				return nil, fmt.Errorf("%w: destination resolution includes a prohibited address", ErrDestination)
			}
		}
		for _, ip := range addresses {
			source := s
			if lan {
				source = InterfaceStatus{}
			}
			conn, e := dialWithInterface(ctx, network, net.JoinHostPort(ip.String(), port), source)
			if e == nil {
				return conn, nil
			}
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
		}
		return nil, errors.New("destination connection failed")
	}
	return &destinationTransport{Transport: t, bound: &boundTransport{base: t}, allowLAN: allowLAN, origin: origin, originLAN: originLAN}
}
