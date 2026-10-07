package netpolicy

import (
	"net/http"
	"net/http/httptest"
	"server/settings"
	"sync/atomic"
	"testing"
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
