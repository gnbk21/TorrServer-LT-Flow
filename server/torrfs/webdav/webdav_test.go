package webdav

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"server/settings"
	"server/web/auth"

	"github.com/gin-gonic/gin"
)

func TestWebDAVUsesConfiguredManagementAuth(t *testing.T) {
	old := settings.HttpAuth
	t.Cleanup(func() { settings.HttpAuth = old })
	gin.SetMode(gin.TestMode)

	for _, tc := range []struct {
		name                   string
		enabled, authenticated bool
		want                   int
	}{
		{"compatible", false, false, http.StatusMethodNotAllowed},
		{"protected", true, false, http.StatusUnauthorized},
		{"authenticated", true, true, http.StatusMethodNotAllowed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			settings.HttpAuth = tc.enabled
			router := gin.New()
			if tc.enabled {
				router.Use(auth.BasicAuth(gin.Accounts{"fixture": "test-only"}))
			}
			MountWebDAV(router)
			// POST is unsupported. Reaching that response requires passing
			// access control, without listing files or starting playback.
			req := httptest.NewRequest(http.MethodPost, "/dav", nil)
			if tc.authenticated {
				req.SetBasicAuth("fixture", "test-only")
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if w.Code != tc.want {
				t.Fatalf("status = %d, want %d", w.Code, tc.want)
			}
		})
	}
}
