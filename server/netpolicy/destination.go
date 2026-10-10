package netpolicy

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"time"
)

var excludedNetworks = []netip.Prefix{
	netip.MustParsePrefix("100.64.0.0/10"), netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("192.0.0.0/24"), netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("198.51.100.0/24"), netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("240.0.0.0/4"), netip.MustParsePrefix("2001:db8::/32"),
}

func publicDestination(ip netip.Addr) bool {
	ip = ip.Unmap()
	if !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
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
		return errors.New("HTTP(S) destination without credentials or fragments required")
	}
	if ip, err := netip.ParseAddr(u.Hostname()); err == nil && !publicDestination(ip) {
		ip = ip.Unmap()
		if !allowLAN || !(ip.IsPrivate() || ip.IsLoopback()) || ip.Zone() != "" {
			return errors.New("destination requires explicit LAN approval")
		}
	}
	return nil
}

// DestinationTransport pins validated resolver answers when dialing. URL-only
// validation cannot prevent DNS rebinding. TLS still verifies the URL hostname.
// Environment proxies are excluded because their resolver would bypass this gate.
func DestinationTransport(allowLAN bool) *http.Transport {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.Proxy = nil
	t.ResponseHeaderTimeout = 10 * time.Second
	t.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, errors.New("invalid destination")
		}
		literal, literalErr := netip.ParseAddr(host)
		addresses, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
		if err != nil {
			return nil, errors.New("destination resolution failed")
		}
		if len(addresses) == 0 {
			return nil, errors.New("destination has no addresses")
		}
		for _, ip := range addresses {
			ip = ip.Unmap()
			approvedLAN := allowLAN && literalErr == nil && literal.Unmap() == ip && (ip.IsPrivate() || ip.IsLoopback()) && ip.Zone() == ""
			if !publicDestination(ip) && !approvedLAN {
				return nil, errors.New("destination resolution includes a prohibited address")
			}
		}
		for _, ip := range addresses {
			d := net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
			conn, e := d.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
			if e == nil {
				return conn, nil
			}
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
		}
		return nil, errors.New("destination connection failed")
	}
	return t
}
