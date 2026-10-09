package web

import (
	"bytes"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"server/settings"
	"server/web/auth"
	"server/web/sslcerts"
	"testing"
)

func TestCertificateAPIRequiresAuthRejectsStaleRevisionAndNeverDownloadsKey(t *testing.T) {
	oldPath, oldSSL, oldAuth, oldArgs, oldSets := settings.Path, settings.Ssl, settings.HttpAuth, settings.Args, settings.BTsets()
	settings.Path = t.TempDir()
	settings.Ssl = true
	settings.HttpAuth = true
	settings.Args = &settings.ExecArgs{}
	t.Cleanup(func() {
		settings.Path, settings.Ssl, settings.HttpAuth, settings.Args = oldPath, oldSSL, oldAuth, oldArgs
		settings.StoreBTsets(oldSets)
	})
	cert, key, err := sslcerts.MakeCertKeyFiles(nil)
	if err != nil {
		t.Fatal(err)
	}
	s := settings.NewDefaultConfig()
	s.SslCert, s.SslKey = cert, key
	settings.StoreBTsets(s)
	if err := os.WriteFile(filepath.Join(settings.Path, "accs.db"), []byte(`{"fixture":"password"}`), 0600); err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	auth.SetupAuth(router)
	setupSSLRoutes(router)
	request := func(method, path string, authorized bool) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, nil)
		if authorized {
			r.SetBasicAuth("fixture", "password")
		}
		r.Header.Set("If-Match", "stale")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		return w
	}
	if got := request("GET", "/ssl/status", false).Code; got != 401 {
		t.Fatal("status accessible without authentication", got)
	}
	before, err := os.ReadFile(cert)
	if err != nil {
		t.Fatal(err)
	}
	if got := request("POST", "/ssl/regenerate", true).Code; got != 409 {
		t.Fatal("stale certificate change accepted", got)
	}
	after, _ := os.ReadFile(cert)
	if !bytes.Equal(before, after) {
		t.Fatal("stale mutation rewrote the identity")
	}
	private, _ := os.ReadFile(key)
	// A user-owned combined PEM may contain key material. Download must extract
	// only certificate blocks, even when the certificate path contains both.
	if err := os.WriteFile(cert, append(before, private...), 0600); err != nil {
		t.Fatal(err)
	}
	w := request(http.MethodGet, "/ssl/cert", true)
	if w.Code != 200 || !bytes.Equal(w.Body.Bytes(), before) || bytes.Contains(w.Body.Bytes(), []byte("PRIVATE KEY")) {
		t.Fatal("certificate download leaked non-public PEM")
	}
}

func TestForceHTTPSMediaModePreservesPathsAndRestrictsControl(t *testing.T) {
	oldPort := settings.SslPort
	settings.SslPort = "443"
	defer func() { settings.SslPort = oldPort }()
	h := forceHTTPSHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }), true)
	for _, tc := range []struct {
		method, path string
		status       int
	}{{"GET", "/stream?link=fixture&play", 204}, {"GET", "/stream?link=fixture&play&save", 307}, {"GET", "/stream?link=fixture&play&preload", 307}, {"POST", "/settings", 307}, {"GET", "/ssl/status", 307}, {"GET", "/flow/play/token", 204}, {"GET", "/gst/hash/probe", 307}, {"GET", "/gst/hash/master.m3u8", 204}} {
		r := httptest.NewRequest(tc.method, "http://[::1]:8090"+tc.path, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != tc.status {
			t.Fatal(tc.path, w.Code, tc.status)
		}
	}
	r := httptest.NewRequest("GET", "http://[::1]:8090/a%2Fb?x=%2F", nil)
	if got := buildHTTPSRedirectTarget(r); got != "https://[::1]/a%2Fb?x=%2F" {
		t.Fatal("escaped redirect changed", got)
	}
}
