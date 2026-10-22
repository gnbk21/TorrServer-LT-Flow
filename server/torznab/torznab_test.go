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
)

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
