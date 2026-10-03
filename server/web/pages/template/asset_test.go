package template

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAssetConditionalRequestsAndUpgrade(t *testing.T) {
	gin.SetMode(gin.TestMode)
	request := func(body string, method, match string) *httptest.ResponseRecorder {
		r := gin.New()
		h := assetHandler([]byte(body), "text/html; charset=utf-8", "no-cache")
		r.GET("/", h); r.HEAD("/", h)
		q := httptest.NewRequest(method, "/", nil)
		q.Header.Set("If-None-Match", match)
		w := httptest.NewRecorder(); r.ServeHTTP(w, q); return w
	}
	first := request("old", http.MethodGet, "")
	etag := first.Header().Get("ETag")
	if first.Code != 200 || len(etag) < 2 || etag[0] != '"' { t.Fatal(first.Code, etag) }
	for _, match := range []string{etag, "W/"+etag, `"different", `+etag, "*"} {
		w := request("old", http.MethodGet, match)
		if w.Code != 304 || w.Body.Len() != 0 { t.Fatalf("conditional %s: %d %q", match,w.Code,w.Body.String()) }
	}
	if w := request("new", http.MethodGet, etag); w.Code != 200 || w.Body.String() != "new" || w.Header().Get("ETag") == etag { t.Fatal("stale entry after upgrade") }
	if w := request("new", http.MethodHead, ""); w.Code != 200 || w.Body.Len() != 0 || w.Header().Get("Content-Length") != "3" { t.Fatal("HEAD does not match GET headers") }
}
