package netpolicy

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"server/settings"
	"sync/atomic"
	"testing"
	"time"
)

func TestUnavailableInterfaceFailsClosedAndCancelsGeneration(t *testing.T) {
	old := settings.BTsets()
	t.Cleanup(func() { settings.StoreBTsets(old); Refresh() })
	f := settings.DefaultFlowSettings()
	settings.StoreBTsets(&settings.BTSets{Flow: f})
	_, before := Refresh()
	f2 := *f
	f2.TorrentInterface = "Flow-missing-test-interface-04917"
	f2.RequireTorrentInterface = true
	settings.StoreBTsets(&settings.BTSets{Flow: &f2})
	s, _ := Refresh()
	if s.State != "WAIT_INTERFACE" || s.DNSVerified || s.LeakProtectionVerified {
		t.Fatal(s)
	}
	select {
	case <-before:
	default:
		t.Fatal("in-flight generation was not cancelled")
	}
	var called atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called.Store(true) }))
	defer server.Close()
	client := &http.Client{Transport: newHTTPTransport()}
	if response, err := client.Get(server.URL); err == nil {
		response.Body.Close()
		t.Fatal("unavailable binding silently used default route")
	}
	if called.Load() {
		t.Fatal("unbound request reached server")
	}
	settings.StoreBTsets(&settings.BTSets{Flow: f})
	response, err := client.Get(server.URL)
	if err != nil {
		t.Fatal("compatible routing did not recover", err)
	}
	response.Body.Close()
}

func TestBoundSourceUsesSelectedAddressAndNeverFallsBack(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	remote := make(chan string, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		remote <- conn.RemoteAddr().(*net.TCPAddr).IP.String()
	}()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	conn, err := dialWithInterface(ctx, "tcp4", listener.Addr().String(), InterfaceStatus{Name: "fixture", State: "BOUND", Addresses: []string{"127.0.0.1"}})
	if err != nil {
		t.Fatal(err)
	}
	conn.Close()
	select {
	case ip := <-remote:
		if ip != "127.0.0.1" {
			t.Fatal(ip)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	for _, status := range []InterfaceStatus{
		{Name: "fixture", State: "WAIT_INTERFACE"},
		{Name: "fixture", State: "BOUND", Addresses: []string{"::1"}},
		{Name: "fixture", State: "BOUND", Addresses: []string{"invalid"}},
	} {
		if connection, err := dialWithInterface(ctx, "tcp4", listener.Addr().String(), status); err == nil {
			connection.Close()
			t.Fatal("selected source fell back unbound", status)
		}
	}
}

func TestInterfaceChangeCancelsResponseBody(t *testing.T) {
	old := settings.BTsets()
	t.Cleanup(func() { settings.StoreBTsets(old); Refresh() })
	f := settings.DefaultFlowSettings()
	settings.StoreBTsets(&settings.BTSets{Flow: f})
	done := make(chan struct{})
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		<-r.Context().Done()
		close(done)
	}))
	defer s.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, s.URL, nil)
	response, err := (&http.Client{Transport: newHTTPTransport()}).Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	f2 := *f
	f2.TorrentInterface = "Flow-missing-test-interface-04917"
	settings.StoreBTsets(&settings.BTSets{Flow: &f2})
	Refresh()
	if _, err := io.ReadAll(response.Body); err == nil {
		t.Fatal("changed generation left a response alive")
	}
	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("old response did not cancel", ctx.Err())
	}
}
