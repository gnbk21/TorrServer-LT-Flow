package api

import (
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"server/diagnostics"
	"server/flow"
	config "server/settings"
	"server/web/auth"
	"strings"
	"testing"
)

func TestMetricsDisabledAndSocketPeerPolicy(t *testing.T) {
	old, oldAuth := config.BTsets(), config.HttpAuth
	t.Cleanup(func() { config.StoreBTsets(old); config.HttpAuth = oldAuth })
	sets := config.NewDefaultConfig()
	config.StoreBTsets(sets)
	router := gin.New()
	router.Use(auth.BasicAuth(gin.Accounts{"monitor": "secret"}))
	router.GET("/metrics", auth.CheckAuth(), metrics)
	request := func(peer string, httpAuth, credentials bool, forwarded string) int {
		config.HttpAuth = httpAuth
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/metrics", nil)
		r.RemoteAddr = peer
		r.Header.Set("X-Forwarded-For", forwarded)
		if credentials {
			r.SetBasicAuth("monitor", "secret")
		}
		router.ServeHTTP(w, r)
		return w.Code
	}
	if got := request("127.0.0.1:1000", false, false, ""); got != 404 {
		t.Fatal("disabled", got)
	}
	sets.Flow.PrometheusEnabled = true
	for _, row := range []struct {
		peer              string
		auth, credentials bool
		xff               string
		want              int
	}{
		{"127.0.0.1:1000", false, false, "", 200}, {"[::1]:1000", false, false, "", 200},
		{"192.0.2.1:1000", false, false, "127.0.0.1", 403}, {"192.0.2.1:1000", true, false, "", 401},
		{"192.0.2.1:1000", true, true, "", 200}, {"127.0.0.1:1000", true, false, "", 401}, {"invalid", false, false, "", 403},
	} {
		if got := request(row.peer, row.auth, row.credentials, row.xff); got != row.want {
			t.Fatal(row, got)
		}
	}
}
func TestMetricsAreCachedAggregateAndMonotonic(t *testing.T) {
	old, oldAuth := config.BTsets(), config.HttpAuth
	t.Cleanup(func() { config.StoreBTsets(old); config.HttpAuth = oldAuth })
	sets := config.NewDefaultConfig()
	sets.Flow.PrometheusEnabled = true
	config.StoreBTsets(sets)
	config.HttpAuth = false
	before := diagnostics.CachedMemory()
	a := &flow.Counters{}
	a.Hit(47)
	a.Wait(500000000)
	router := gin.New()
	router.GET("/metrics", metrics)
	scrape := func() string {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/metrics", nil)
		r.RemoteAddr = "127.0.0.1:1000"
		router.ServeHTTP(w, r)
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
		return w.Body.String()
	}
	first := scrape()
	totals := flow.Totals()
	_ = &flow.Counters{}
	second := scrape()
	if flow.Totals() != totals {
		t.Fatal("scrape mutated measurements")
	}
	if before != diagnostics.CachedMemory() {
		t.Fatal("scrape initiated memory sampling")
	}
	if first != second || !strings.Contains(first, "flow_cache_hit_bytes_total") || !strings.Contains(first, "flow_read_wait_seconds_total") || !strings.Contains(first, `source="torznab"`) {
		t.Fatal("invalid exposition", first)
	}
	for _, secret := range []string{"filename=", "hash=", "query=", "address=", "token="} {
		if strings.Contains(first, secret) {
			t.Fatal("identity label", secret)
		}
	}
}
