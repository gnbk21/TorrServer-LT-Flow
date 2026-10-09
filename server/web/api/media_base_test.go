package api

import (
	"crypto/tls"
	"net/http/httptest"
	"testing"

	"github.com/gin-contrib/location/v2"
	"github.com/gin-gonic/gin"
	"server/settings"
	"server/web/sslcerts"
)

func TestExternalMediaBaseRespectsHTTPSModesAndIPv6(t *testing.T) {
	oldPath, oldSSL, oldPort, oldArgs, oldSets := settings.Path, settings.Ssl, settings.Port, settings.Args, settings.BTsets()
	settings.Path, settings.Ssl, settings.Port = t.TempDir(), true, "8090"
	t.Cleanup(func() {
		settings.Path, settings.Ssl, settings.Port, settings.Args = oldPath, oldSSL, oldPort, oldArgs
		settings.StoreBTsets(oldSets)
	})
	cert, key, err := sslcerts.MakeCertKeyFiles(nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		args settings.ExecArgs
		user bool
		want string
	}{
		{"both", settings.ExecArgs{}, false, "http://[fd00::1]:8090"},
		{"only", settings.ExecArgs{HTTPSOnly: true}, false, "https://[fd00::1]:8091"},
		{"redirect", settings.ExecArgs{ForceHTTPS: true}, false, "https://[fd00::1]:8091"},
		{"media", settings.ExecArgs{ForceHTTPS: true, HTTPMedia: true}, false, "http://[fd00::1]:8090"},
		{"user-identity", settings.ExecArgs{}, true, "https://[fd00::1]:8091"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			settings.Args = &tc.args
			s := settings.NewDefaultConfig()
			s.SslCert, s.SslKey = cert, key
			if tc.user {
				s.SslCert, s.SslKey = "user.crt", "user.key"
			}
			settings.StoreBTsets(s)
			router := gin.New()
			router.Use(location.Default())
			router.GET("/mediabase", mediaBase)
			r := httptest.NewRequest("GET", "https://[fd00::1]:8091/mediabase", nil)
			r.TLS = &tls.ConnectionState{}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)
			if w.Code != 200 || w.Body.String() != `{"base":"`+tc.want+`"}` || w.Header().Get("Cache-Control") != "no-store" {
				t.Fatal(w.Code, w.Body.String(), w.Header())
			}
		})
	}
}
