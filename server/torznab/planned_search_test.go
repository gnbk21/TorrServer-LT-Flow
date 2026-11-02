package torznab

import (
	"context"
	"net/http"
	"net/http/httptest"
	"server/settings"
	"sync/atomic"
	"testing"
	"time"
)

func TestStructuredSearchFallsBackOnlyOnce(t *testing.T) {
	for _, response := range []struct {
		name, body string
		status     int
	}{{"empty", `<rss><channel/></rss>`, 200}, {"unsupported_xml", `<error code="203" description="unsupported parameter"/>`, 200}, {"unsupported_http", "unsupported", 422}} {
		t.Run(response.name, func(t *testing.T) {
			var structured, generic atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("t") == "movie" {
					structured.Add(1)
					w.WriteHeader(response.status)
					_, _ = w.Write([]byte(response.body))
					return
				}
				generic.Add(1)
				_, _ = w.Write([]byte(`<rss><channel/></rss>`))
			}))
			defer server.Close()
			rememberCaps(server.URL, "key", &Caps{Searching: Searching{Movie: SearchCapability{Available: "yes", SupportedParams: "q,imdbid"}}})
			_, mode, err := searchPlanned(context.Background(), server.URL, "key", Query{Text: "Fixture", Type: "movie", IMDb: "tt123"}, "Fixture", "", 0, 10)
			if err != nil || mode != "generic_fallback" || structured.Load() != 1 || generic.Load() != 1 {
				t.Fatal(mode, err, structured.Load(), generic.Load())
			}
		})
	}
}
func TestColdSearchDoesNotAwaitCapabilitiesAndCancellationStopsFallback(t *testing.T) {
	capsStarted := make(chan struct{}, 1)
	release := make(chan struct{})
	var generic atomic.Int32
	var prematureDiscovery atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("t") == "caps" {
			if generic.Load() < 4 {
				prematureDiscovery.Store(true)
			}
			select {
			case capsStarted <- struct{}{}:
			default:
			}
			select {
			case <-release:
			case <-r.Context().Done():
			}
			_, _ = w.Write([]byte(`<caps/>`))
			return
		}
		generic.Add(1)
		_, _ = w.Write([]byte(`<rss><channel/></rss>`))
	}))
	defer server.Close()
	defer close(release)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	old := settings.BTsets()
	config := settings.NewDefaultConfig()
	config.EnableTorznabSearch = true
	for _, key := range []string{"cold1", "cold2", "cold3", "cold4"} {
		config.TorznabUrls = append(config.TorznabUrls, settings.TorznabConfig{Host: server.URL, Key: key})
	}
	settings.StoreBTsets(config)
	defer settings.StoreBTsets(old)
	_, outcomes := SearchDetailedQuery(ctx, Query{Text: "Fixture", Type: "movie"}, -1, "", 0, 10)
	if len(outcomes) != 4 || generic.Load() != 4 || prematureDiscovery.Load() {
		t.Fatal(outcomes, generic.Load(), prematureDiscovery.Load())
	}
	for _, outcome := range outcomes {
		if outcome.QueryMode != "generic" || outcome.State != "empty" {
			t.Fatal(outcome)
		}
	}
	select {
	case <-capsStarted:
	case <-time.After(time.Second):
		t.Fatal("no bounded discovery")
	}
	rememberCaps(server.URL, "warm", &Caps{Searching: Searching{Movie: SearchCapability{Available: "yes", SupportedParams: "q"}}})
	stopped, stop := context.WithCancel(context.Background())
	stop()
	before := generic.Load()
	_, _, err := searchPlanned(stopped, server.URL, "warm", Query{Text: "Fixture", Type: "movie"}, "Fixture", "", 0, 10)
	if err == nil || generic.Load() != before {
		t.Fatal("cancellation launched a fallback", err)
	}
}

func TestGenericStructuredParametersStillReceiveOnlyOneTitleFallback(t *testing.T) {
	var structured, generic atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("year") != "" {
			structured.Add(1)
		} else {
			generic.Add(1)
		}
		_, _ = w.Write([]byte(`<rss><channel/></rss>`))
	}))
	defer server.Close()
	rememberCaps(server.URL, "key", &Caps{Searching: Searching{Search: SearchCapability{Available: "yes", SupportedParams: "q,year"}}})
	_, mode, err := searchPlanned(context.Background(), server.URL, "key", Query{Text: "Fixture", Year: "2026"}, "Fixture", "", 0, 10)
	if err != nil || mode != "generic_fallback" || structured.Load() != 1 || generic.Load() != 1 {
		t.Fatal(mode, err, structured.Load(), generic.Load())
	}
}
