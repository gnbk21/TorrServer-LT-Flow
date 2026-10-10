// Package netpolicy implements optional source-interface binding. It is not an
// OS firewall or a claim that the system DNS resolver cannot leak.
package netpolicy

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"server/settings"
	"sort"
	"strings"
	"sync"
	"time"
)

type InterfaceStatus struct {
	Name                   string   `json:"name"`
	State                  string   `json:"state"`
	Addresses              []string `json:"addresses"`
	Required               bool     `json:"required"`
	DNSVerified            bool     `json:"dns_verified"`
	LeakProtectionVerified bool     `json:"leak_protection_verified"`
}

var interfaceLifecycle struct {
	sync.Mutex
	fingerprint string
	changed     chan struct{}
}

func Snapshot() InterfaceStatus {
	f := settings.CurrentFlow()
	s := InterfaceStatus{Name: f.TorrentInterface, Required: f.RequireTorrentInterface, State: "DISABLED", Addresses: []string{}}
	if s.Name == "" {
		return s
	}
	s.State = "WAIT_INTERFACE"
	iface, err := net.InterfaceByName(s.Name)
	if err != nil || iface.Flags&net.FlagUp == 0 {
		return s
	}
	addresses, err := iface.Addrs()
	if err != nil {
		return s
	}
	for _, a := range addresses {
		ip, _, err := net.ParseCIDR(a.String())
		current := settings.BTsets()
		ipv6 := current != nil && current.EnableIPv6
		if err == nil && ip.IsGlobalUnicast() && !ip.IsLinkLocalUnicast() && (ip.To4() != nil || ipv6) {
			s.Addresses = append(s.Addresses, ip.String())
		}
	}
	sort.Strings(s.Addresses)
	if len(s.Addresses) > 0 {
		s.State = "BOUND"
	}
	return s
}

// Refresh cancels Go torrent fetches when the selected interface changes. Native
// recovery is reconciled by BTServer's existing lifecycle, without a restart.
func Refresh() (InterfaceStatus, <-chan struct{}) {
	s := Snapshot()
	fingerprint := s.Name + ":" + s.State + ":" + strings.Join(s.Addresses, ",")
	interfaceLifecycle.Lock()
	defer interfaceLifecycle.Unlock()
	if interfaceLifecycle.changed == nil || interfaceLifecycle.fingerprint != fingerprint {
		if interfaceLifecycle.changed != nil {
			close(interfaceLifecycle.changed)
		}
		interfaceLifecycle.changed = make(chan struct{})
		interfaceLifecycle.fingerprint = fingerprint
	}
	return s, interfaceLifecycle.changed
}

type boundTransport struct {
	base       *http.Transport
	mu         sync.Mutex
	generation <-chan struct{}
}

// dialWithInterface never falls back to an unbound socket after a source was
// selected. Literal destinations avoid resolving the pinned address a second time.
func dialWithInterface(ctx context.Context, network, address string, s InterfaceStatus) (net.Conn, error) {
	d := net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	if s.Name == "" {
		return d.DialContext(ctx, network, address)
	}
	if s.State != "BOUND" {
		return nil, errors.New("selected torrent interface is unavailable")
	}
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return nil, errors.New("invalid destination")
	}
	destination := net.ParseIP(host)
	for _, text := range s.Addresses {
		ip := net.ParseIP(text)
		if ip == nil || (destination != nil && (ip.To4() != nil) != (destination.To4() != nil)) {
			continue
		}
		d.LocalAddr = &net.TCPAddr{IP: ip}
		connection, err := d.DialContext(ctx, network, address)
		if err == nil {
			return connection, nil
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
	}
	return nil, errors.New("torrent fetch could not use selected interface")
}

var transportOnce sync.Once
var sharedTransport http.RoundTripper

func HTTPTransport() http.RoundTripper {
	transportOnce.Do(func() { sharedTransport = newHTTPTransport() })
	return sharedTransport
}
func newHTTPTransport() http.RoundTripper {
	t := http.DefaultTransport.(*http.Transport).Clone()
	// An environment proxy would own its own route and DNS policy.
	t.Proxy = func(req *http.Request) (*url.URL, error) {
		if Snapshot().Name != "" {
			return nil, nil
		}
		return http.ProxyFromEnvironment(req)
	}
	t.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		return dialWithInterface(ctx, network, address, Snapshot())
	}
	return &boundTransport{base: t}
}
func (t *boundTransport) CloseIdleConnections() { t.base.CloseIdleConnections() }
func (t *boundTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	s, changed := Refresh()
	// Close old pooled sockets synchronously before this request can reuse them.
	// The cancellation watcher alone can race a request on the new generation.
	t.mu.Lock()
	if t.generation != changed {
		t.base.CloseIdleConnections()
		t.generation = changed
	}
	t.mu.Unlock()
	if s.Name != "" && s.State != "BOUND" {
		t.base.CloseIdleConnections()
		return nil, errors.New("selected torrent interface is unavailable")
	}
	ctx, cancel := context.WithCancel(req.Context())
	go func() {
		select {
		case <-changed:
			cancel()
			t.base.CloseIdleConnections()
		case <-ctx.Done():
		}
	}()
	response, err := t.base.RoundTrip(req.Clone(ctx))
	if err != nil {
		cancel()
		return nil, err
	}
	response.Body = &cancelBody{ReadCloser: response.Body, cancel: cancel}
	return response, nil
}
