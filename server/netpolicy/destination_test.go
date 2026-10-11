package netpolicy

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"server/settings"
	"strings"
	"testing"
)

func TestProxyDestinationPolicy(t *testing.T) {
	for _, raw := range []string{"http://127.0.0.1/", "http://[::1]/", "http://[::ffff:127.0.0.1]/", "http://169.254.169.254/", "http://100.64.0.1/", "http://192.168.1.1/", "file:///etc/passwd", "https://user:secret@example.com/"} {
		u, _ := url.Parse(raw)
		if ValidateDestination(u, false) == nil {
			t.Fatal("allowed", raw)
		}
	}
	u, _ := url.Parse("http://169.254.169.254/")
	if ValidateDestination(u, true) == nil {
		t.Fatal("LAN approval allowed link-local metadata")
	}
}

func TestProxyBindingFailsClosedButPreservesExplicitLAN(t *testing.T) {
	old := settings.BTsets()
	t.Cleanup(func() { settings.StoreBTsets(old); Refresh() })
	f := settings.DefaultFlowSettings()
	f.TorrentInterface = "Flow-missing-test-interface-04917"
	f.RequireTorrentInterface = true
	settings.StoreBTsets(&settings.BTSets{Flow: f})
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "local") }))
	defer s.Close()
	for _, allow := range []bool{false, true} {
		transport := DestinationTransport(allow)
		defer transport.CloseIdleConnections()
		if _, err := transport.DialContext(context.Background(), "tcp", "8.8.8.8:443"); err == nil || !strings.Contains(err.Error(), "interface is unavailable") {
			t.Fatal("public dialing did not fail before DNS or an unbound connection", err)
		}
		response, err := (&http.Client{Transport: transport}).Get(s.URL)
		if !allow {
			if err == nil {
				response.Body.Close()
				t.Fatal("LAN was implicitly approved")
			}
			continue
		}
		if err != nil {
			t.Fatal("explicit literal LAN approval was lost", err)
		}
		data, err := io.ReadAll(response.Body)
		response.Body.Close()
		if err != nil || string(data) != "local" {
			t.Fatal("LAN response failed", err)
		}
	}
}
func TestProxyDialRejectsLocalDNSAndAllowsOnlyExplicitLiteral(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "ok") }))
	defer s.Close()
	for _, allow := range []bool{false, true} {
		transport := DestinationTransport(allow)
		defer transport.CloseIdleConnections()
		client := &http.Client{Transport: transport}
		r, err := client.Get(s.URL)
		if !allow {
			if err == nil {
				r.Body.Close()
				t.Fatal("default reached LAN")
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		r.Body.Close()
		u, _ := url.Parse(s.URL)
		if _, err = transport.DialContext(context.Background(), "tcp", "localhost:"+u.Port()); err == nil {
			t.Fatal("LAN permission was extended to DNS hostnames")
		}
	}
}
