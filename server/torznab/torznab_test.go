package torznab

import (
	"context"
	"net/http"
	"net/http/httptest"
	"server/settings"
	"server/version"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// MatriX.148 #947: check the real result mapping for both XML size sources.
func TestSearchOneFormatsSizeInBinaryUnits(t *testing.T) {
	previous := settings.BTsets()
	settings.StoreBTsets(&settings.BTSets{Flow: settings.DefaultFlowSettings()})
	t.Cleanup(func() { settings.StoreBTsets(previous) })
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<rss><channel>
<item><title>enclosure</title><enclosure url="magnet:?xt=urn:btih:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" length="1610612736"/></item>
<item><title>bytes</title><size>512</size></item>
<item><title>kibibytes</title><size>1024</size></item>
<item><title>mebibytes</title><size>734003200</size></item>
<item><title>tebibytes</title><size>1099511627776</size></item>
</channel></rss>`))
	}))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	results, err := searchOneDetailed(ctx, srv.URL, "fixture", "generated", "fixture", "", 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"1.5 GiB", "512 B", "1.0 KiB", "700.0 MiB", "1.0 TiB"}
	if len(results) != len(want) {
		t.Fatalf("got %d results, want %d", len(results), len(want))
	}
	for i, size := range want {
		if results[i].Size != size {
			t.Errorf("result %d size=%q, want %q", i, results[i].Size, size)
		}
	}
}

func TestDetailedSearchBoundsParallelRequestsAndKeepsSourceOutcomes(t *testing.T) {
	old := settings.BTsets()
	defer settings.StoreBTsets(old)
	var active, peak atomic.Int32
	seen := make(chan struct{}, 8)
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		now := active.Add(1)
		defer active.Add(-1)
		for previous := peak.Load(); now > previous && !peak.CompareAndSwap(previous, now); previous = peak.Load() {
		}
		seen <- struct{}{}
		select {
		case <-release:
		case <-r.Context().Done():
			return
		}
		if r.URL.Query().Get("apikey") == "bad" {
			w.WriteHeader(401)
			return
		}
		_, _ = w.Write([]byte(`<rss><channel><item><title>generated</title><link>magnet:?xt=urn:btih:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa</link></item></channel></rss>`))
	}))
	defer srv.Close()
	configs := []settings.TorznabConfig{}
	for i := 0; i < 6; i++ {
		key := "good"
		if i == 5 {
			key = "bad"
		}
		configs = append(configs, settings.TorznabConfig{Host: srv.URL, Key: key})
	}
	settings.StoreBTsets(&settings.BTSets{EnableTorznabSearch: true, TorznabUrls: configs, Flow: settings.DefaultFlowSettings()})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	done := make(chan struct{})
	var rows int
	var states []string
	go func() {
		result, statuses := SearchDetailed(ctx, "generated", -1, "", 0, 100)
		rows = len(result)
		for _, status := range statuses {
			states = append(states, status.State)
		}
		close(done)
	}()
	for i := 0; i < 4; i++ {
		select {
		case <-seen:
		case <-ctx.Done():
			close(release)
			t.Fatal("parallel workers did not start")
		}
	}
	if active.Load() != 4 {
		close(release)
		t.Fatal("expected four outgoing requests")
	}
	close(release)
	<-done
	if peak.Load() > 4 || rows != 5 || len(states) != 6 || states[5] != "auth_required" {
		t.Fatalf("wrong parallel outcome: %d %d %v", peak.Load(), rows, states)
	}
}

func TestCapsDoesNotBypassUnavailableInterface(t *testing.T) {
	previous := settings.BTsets()
	flow := settings.DefaultFlowSettings()
	flow.TorrentInterface, flow.RequireTorrentInterface = "Flow-missing-torznab-interface-418", true
	settings.StoreBTsets(&settings.BTSets{Flow: flow})
	t.Cleanup(func() { settings.StoreBTsets(previous) })
	var called atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called.Store(true) }))
	defer srv.Close()
	if _, err := FetchCaps(context.Background(), srv.URL, ""); err == nil || !strings.Contains(err.Error(), "interface is unavailable") {
		t.Fatalf("binding was bypassed: %v", err)
	}
	if called.Load() {
		t.Fatal("indexer request bypassed the selected interface")
	}
}

func TestNormalizeHost(t *testing.T) {
	tests := []struct {
		name    string
		host    string
		want    string
		wantErr bool
	}{
		{
			name: "jackett torznab base",
			host: "http://192.168.1.10:9117/api/v2.0/indexers/all/results/torznab/",
			want: "http://192.168.1.10:9117/api/v2.0/indexers/all/results/torznab/api",
		},
		{
			name: "already ends with api",
			host: "http://192.168.1.10:9117/api/v2.0/indexers/all/results/torznab/api",
			want: "http://192.168.1.10:9117/api/v2.0/indexers/all/results/torznab/api",
		},
		{
			name: "already ends with api slash",
			host: "http://192.168.1.10:9117/api/v2.0/indexers/all/results/torznab/api/",
			want: "http://192.168.1.10:9117/api/v2.0/indexers/all/results/torznab/api",
		},
		{
			name: "prowlarr indexer id",
			host: "http://localhost:9696/1",
			want: "http://localhost:9696/1/api",
		},
		{
			name: "scheme default",
			host: "localhost:9117/api/v2.0/indexers/all/results/torznab/",
			want: "http://localhost:9117/api/v2.0/indexers/all/results/torznab/api",
		},
		{
			name:    "empty",
			host:    "   ",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := normalizeHost(tt.host)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %v", u)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := u.String(); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTorznabIdentifiesBothSearchAndCapabilitiesRequests(t *testing.T) {
	previous := settings.BTsets()
	settings.StoreBTsets(&settings.BTSets{Flow: settings.DefaultFlowSettings()})
	t.Cleanup(func() { settings.StoreBTsets(previous) })
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path != "/indexer/api" || r.Header.Get("User-Agent") != "TorrServer-Flow-Torznab/"+version.Version {
			t.Error("wrong endpoint or missing product identity")
		}
		if r.URL.Query().Get("t") == "caps" {
			_, _ = w.Write([]byte(`<caps><limits max="100" default="50"/></caps>`))
		} else {
			_, _ = w.Write([]byte(`<rss><channel/></rss>`))
		}
	}))
	defer srv.Close()
	searchOne(context.Background(), srv.URL+"/indexer/api/", "fixture", "episode", "fixture", "", 0, 0)
	if _, err := FetchCaps(context.Background(), srv.URL+"/indexer/api/", "fixture"); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 2 {
		t.Fatal("search or capabilities request missing")
	}
}

func TestSearchOneTakesTrackerFromIndexerTags(t *testing.T) {
	const feed = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:torznab="http://torznab.com/schemas/2015/feed">
<channel>
<item><title>A</title><link>magnet:?xt=urn:btih:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa</link><jackettindexer id="rutracker">RuTracker.org</jackettindexer></item>
<item><title>B</title><link>magnet:?xt=urn:btih:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb</link><prowlarrindexer id="7">Kinozal</prowlarrindexer></item>
<item><title>C</title><link>magnet:?xt=urn:btih:cccccccccccccccccccccccccccccccccccccccc</link></item>
</channel>
</rss>`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(feed))
	}))
	defer srv.Close()

	got := searchOne(context.Background(), srv.URL, "key", "q", "Jackett", "", 0, 0)
	want := []string{"RuTracker.org", "Kinozal", "Jackett"}
	if len(got) != len(want) {
		t.Fatalf("got %d results, want %d", len(got), len(want))
	}
	for i, w := range want {
		if got[i].Tracker != w {
			t.Errorf("result %d tracker = %q, want %q", i, got[i].Tracker, w)
		}
	}
}
