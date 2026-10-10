package netpolicy

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
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
