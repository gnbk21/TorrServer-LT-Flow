package netpolicy

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestFetchDestinationsAndRedirectGrants(t *testing.T) {
	var reached atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { reached.Add(1); w.WriteHeader(204) }))
	defer target.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, 302) }))
	defer source.Close()
	if _, err := FetchClient(target.URL, false, time.Second); err == nil {
		t.Fatal("public fetch granted LAN")
	}
	client, err := FetchClient(source.URL, true, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer client.CloseIdleConnections()
	if response, err := client.Get(source.URL); err == nil {
		response.Body.Close()
		t.Fatal("redirect inherited another origin's LAN grant")
	}
	if reached.Load() != 0 {
		t.Fatal("redirect reached a private destination")
	}
	for _, endpoint := range []string{"http://169.254.169.254/", "http://100.64.0.1/", "http://[64:ff9b::a9fe:a9fe]/", "file:///owned", "https://user:pass@example.com/", "http://[fe80::1%25eth0]/"} {
		if _, err := FetchClient(endpoint, true, time.Second); err == nil {
			t.Fatal("unsafe configured target", endpoint)
		}
	}
	client, err = FetchClient(target.URL, true, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer client.CloseIdleConnections()
	response, err := client.Get(target.URL)
	if err != nil {
		t.Fatal("explicit local indexer failed", err)
	}
	response.Body.Close()
	localHostURL := strings.Replace(target.URL, "127.0.0.1", "localhost", 1)
	publicLocal, err := FetchClient(localHostURL, false, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer publicLocal.CloseIdleConnections()
	if r, err := publicLocal.Get(localHostURL); !errors.Is(err, ErrDestination) {
		if r != nil {
			r.Body.Close()
		}
		t.Fatal("public DNS destination reached loopback", err)
	}
	localClient, err := FetchClient(localHostURL, true, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer localClient.CloseIdleConnections()
	localResponse, err := localClient.Get(localHostURL)
	if err != nil {
		t.Fatal("configured localhost failed", err)
	}
	localResponse.Body.Close()
	changedScheme, _ := url.Parse(strings.Replace(localHostURL, "http:", "https:", 1))
	if _, err := localClient.Transport.RoundTrip(&http.Request{Method: "GET", URL: changedScheme, Header: http.Header{}}); !errors.Is(err, ErrDestination) {
		t.Fatal("localhost grant escaped its scheme", err)
	}
	public, err := FetchClient("https://example.com", false, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer public.CloseIdleConnections()
	u, _ := url.Parse("http://example.com/")
	if err := public.CheckRedirect(&http.Request{URL: u}, nil); err == nil {
		t.Fatal("HTTPS downgrade accepted")
	}
	// Public hostnames get no DNS-to-loopback grant, even for configured sources.
	public, err = FetchClient("http://untrusted.example", true, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer public.CloseIdleConnections()
	transport := public.Transport.(*destinationTransport)
	if transport.originLAN || transport.Proxy != nil {
		t.Fatal("public host acquired LAN permission or an environment proxy")
	}
}
