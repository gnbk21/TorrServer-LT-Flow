package api

import (
	"net/http"
	"net/http/httptest"
	sets "server/settings"
	"testing"

	"github.com/gin-gonic/gin"
)

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
		req := httptest.NewRequest(method, "http://dashboard.test"+path, nil)
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
	if got := request("POST", "/settings", "http://dashboard.test", "192.0.2.1:9"); got != 204 {
		t.Fatal(got)
	}
	if got := request("POST", "/settings", "http://dashboard.test", "192.0.2.1:9"); got != 429 {
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
