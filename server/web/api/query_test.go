package api

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	sets "server/settings"
)

func TestSearchHandlersPreserveQueryValues(t *testing.T) {
	old := sets.BTsets()
	t.Cleanup(func() { sets.StoreBTsets(old) })
	seen := make(chan string, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "v1.0") {
			seen <- r.URL.Query().Get("search")
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("[]"))
		} else {
			seen <- r.URL.Query().Get("q")
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte("<rss><channel/></rss>"))
		}
	}))
	defer upstream.Close()
	sets.StoreBTsets(&sets.BTSets{EnableTorznabSearch: true, EnableJacRedSearch: true, JacRedUrl: upstream.URL, TorznabUrls: []sets.TorznabConfig{{Host: upstream.URL, Key: "fixture"}}, Flow: sets.DefaultFlowSettings()})
	for _, handler := range []gin.HandlerFunc{torznabSearch, jacredSearch} {
		for _, tc := range []struct{ input, want string }{{"100% Pure", "100% Pure"}, {"C++", "C++"}, {"Сериал + кино", "Сериал + кино"}, {"Caf%C3%A9", "Café"}, {"%zz", "%zz"}} {
			response := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(response)
			ctx.Request = httptest.NewRequest(http.MethodGet, "/?"+url.Values{"query": {tc.input}}.Encode(), nil)
			handler(ctx)
			if response.Code != http.StatusOK {
				t.Fatalf("search rejected %q: %d", tc.input, response.Code)
			}
			select {
			case got := <-seen:
				if got != tc.want {
					t.Fatalf("indexer received %q, want %q", got, tc.want)
				}
			default:
				t.Fatal("search did not reach the local indexer")
			}
		}
	}
}

func TestStreamHandlersPreserveEncodedSourceComponents(t *testing.T) {
	old := sets.BTsets()
	sets.StoreBTsets(&sets.BTSets{Flow: sets.DefaultFlowSettings()})
	t.Cleanup(func() { sets.StoreBTsets(old) })
	seen := make(chan string, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen <- r.URL.RequestURI()
		w.WriteHeader(http.StatusNotFound) // stop before adding a torrent
	}))
	defer upstream.Close()
	path := "/a%2Fb.torrent?token=A%2BB+Z&sig=%252F"
	source := upstream.URL + path
	for _, handler := range []gin.HandlerFunc{stream, streamNoAuth} {
		for _, value := range []string{source, url.PathEscape(source)} {
			response := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(response)
			ctx.Request = httptest.NewRequest(http.MethodGet, "/stream?"+url.Values{"link": {value}}.Encode(), nil)
			handler(ctx)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("expected fixture download rejection: %d", response.Code)
			}
			select {
			case got := <-seen:
				if got != path {
					t.Fatalf("source URL changed: %q, want %q", got, path)
				}
			default:
				t.Fatal("handler did not reach the local source")
			}
		}
	}
	for _, value := range []string{source, url.PathEscape(source), "file:///owned.torrent", url.PathEscape("file:///owned.torrent")} {
		response := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(response)
		ctx.Request = httptest.NewRequest(http.MethodGet, "/stream?"+url.Values{"link": {value}, "play": {""}}.Encode(), nil)
		ctx.Set("auth_required", true)
		stream(ctx)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("unauthenticated source accepted: %d", response.Code)
		}
		select {
		case <-seen:
			t.Fatal("unauthenticated source was fetched")
		default:
		}
	}
}
