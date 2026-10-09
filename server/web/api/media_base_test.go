package api

import (
	"crypto/tls"
	"net/http/httptest"
	"testing"

	"github.com/gin-contrib/location/v2"
	"github.com/gin-gonic/gin"
	config "server/settings"
	"server/web/sslcerts"
)

func TestExternalMediaBaseRespectsHTTPSModesAndIPv6(t *testing.T) {
	oldPath, oldSSL, oldPort, oldArgs, oldSets := config.Path, config.Ssl, config.Port, config.Args, config.BTsets()
	config.Path, config.Ssl, config.Port = t.TempDir(), true, "8090"
	t.Cleanup(func() {
		config.Path, config.Ssl, config.Port, config.Args = oldPath, oldSSL, oldPort, oldArgs
		config.StoreBTsets(oldSets)
	})
	cert, key, err := sslcerts.MakeCertKeyFiles(nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		args config.ExecArgs
		user bool
		want string
	}{
		{"both", config.ExecArgs{}, false, "http://[fd00::1]:8090"},
		{"only", config.ExecArgs{HTTPSOnly: true}, false, "https://[fd00::1]:8091"},
		{"redirect", config.ExecArgs{ForceHTTPS: true}, false, "https://[fd00::1]:8091"},
		{"media", config.ExecArgs{ForceHTTPS: true, HTTPMedia: true}, false, "http://[fd00::1]:8090"},
		{"user-identity", config.ExecArgs{}, true, "https://[fd00::1]:8091"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config.Args = &tc.args
			s := config.NewDefaultConfig()
			s.SslCert, s.SslKey = cert, key
			if tc.user {
				s.SslCert, s.SslKey = "user.crt", "user.key"
			}
			config.StoreBTsets(s)
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
