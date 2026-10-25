package api

import (
	"net/http"
	"net/http/httptest"
	sets "server/settings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCompatibleManagementStillRejectsUnapprovedBrowserOrigins(t *testing.T) {
	old := sets.BTsets()
	t.Cleanup(func() { sets.StoreBTsets(old) })
	sets.StoreBTsets(&sets.BTSets{Flow: sets.DefaultFlowSettings()})
	r := gin.New()
	r.Use(ManagementPolicy())
	r.Any("/settings", func(c *gin.Context) { c.Status(204) })
	r.GET("/shutdown", func(c *gin.Context) { c.Status(204) })
	r.GET("/stream", func(c *gin.Context) { c.Status(204) })
	for _, tc := range []struct {
		path, origin, site string
		want               int
	}{
		{"/settings", "https://evil.example", "cross-site", 403},
		{"/settings", "null", "cross-site", 403},
		{"/settings", "http://127.0.0.1:8090", "same-origin", 204},
		{"/settings", "", "", 204}, // native Lampa retains its auth path
		{"/shutdown", "", "cross-site", 403},
		{"/shutdown", "", "same-site", 403},
		{"/stream", "https://player.example", "cross-site", 204},
		{"/stream?link=http%3A%2F%2F127.0.0.1%2Finternal", "https://evil.example", "cross-site", 403},
		{"/stream?link=file%3A%2F%2F%2Fprivate.torrent", "https://evil.example", "cross-site", 403},
		{"/stream?link=magnet%3A%3Fxt%3Durn%3Abtih%3Aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "https://player.example", "cross-site", 204},
		{"/stream?save&link=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "https://evil.example", "cross-site", 403},
	} {
		request := httptest.NewRequest("GET", "http://127.0.0.1:8090"+tc.path, nil)
		request.Header.Set("Origin", tc.origin)
		request.Header.Set("Sec-Fetch-Site", tc.site)
		response := httptest.NewRecorder()
		r.ServeHTTP(response, request)
		if response.Code != tc.want {
			t.Fatalf("%+v: %d", tc, response.Code)
		}
	}
}

func TestBrowserManagementRejectsRebindingButAllowsApprovedProxy(t *testing.T) {
	old := sets.BTsets()
	t.Cleanup(func() { sets.StoreBTsets(old) })
	f := sets.DefaultFlowSettings()
	sets.StoreBTsets(&sets.BTSets{Flow: f})
	r := gin.New()
	r.Use(ManagementPolicy())
	r.GET("/settings", func(c *gin.Context) { c.Status(204) })
	request := func() int {
		req := httptest.NewRequest("GET", "http://proxy.example/settings", nil)
		req.Header.Set("Origin", "http://proxy.example")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Code
	}
	if got := request(); got != 403 {
		t.Fatal("arbitrary DNS host accepted", got)
	}
	next := *f
	next.ManagementOrigins = "http://proxy.example"
	sets.StoreBTsets(&sets.BTSets{Flow: &next})
	if got := request(); got != 204 {
		t.Fatal("approved reverse proxy refused", got)
	}
}

func TestManagementOriginRateAndPlaybackCompatibility(t *testing.T) {
	old := sets.BTsets()
	t.Cleanup(func() { sets.StoreBTsets(old) })
	f := sets.DefaultFlowSettings()
	f.SecurityProfile = "restricted"
	f.ManagementOrigins = "https://controller.example"
	f.ManagementRateLimit = 1
	sets.StoreBTsets(&sets.BTSets{Flow: f})
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(ManagementPolicy(), PlaybackPolicy())
	r.Any("/settings", func(c *gin.Context) { c.Status(204) })
	r.Any("/stream", func(c *gin.Context) { c.Status(204) })
	request := func(method, path, origin, remote string) int {
		req := httptest.NewRequest(method, "http://127.0.0.1:8090"+path, nil)
		req.RemoteAddr = remote
		req.Header.Set("Origin", origin)
		req.Header.Set("X-Forwarded-For", "203.0.113.7")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Code
	}
	if got := request("OPTIONS", "/settings", "https://evil.example", "192.0.2.1:9"); got != 403 {
		t.Fatal(got)
	}
	if got := request("OPTIONS", "/settings", "https://controller.example", "192.0.2.1:9"); got != 204 {
		t.Fatal(got)
	}
	if got := request("POST", "/settings", "http://127.0.0.1:8090", "192.0.2.1:9"); got != 204 {
		t.Fatal(got)
	}
	if got := request("POST", "/settings", "http://127.0.0.1:8090", "192.0.2.1:9"); got != 429 {
		t.Fatal(got)
	}
	if got := request("POST", "/settings", "", "192.0.2.2:9"); got != 204 {
		t.Fatal("forwarded header affected identity", got)
	}
	if got := request("GET", "/stream", "https://evil.example", "192.0.2.1:9"); got != 204 {
		t.Fatal("compatible playback changed", got)
	}
	f2 := *f
	f2.RequirePlaybackToken = true
	sets.StoreBTsets(&sets.BTSets{Flow: &f2})
	if got := request(http.MethodGet, "/stream?stat=ffprobe", "", "127.0.0.1:9"); got != 401 {
		t.Fatal("public probe marker bypassed access", got)
	}
}
