package api

import (
	"net/http"
	"net/http/httptest"
	"server/settings"
	"server/web/auth"
	"sync/atomic"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCredentialFreePlayRejectsRemoteAndLocalSourcesBeforeParsing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var fetched atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fetched.Add(1)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer upstream.Close()
	for _, source := range []string{upstream.URL, "file:///nonexistent-owned.torrent",
		"https%3A%2F%2Fsource.example%2Fmetadata.torrent"} {
		response := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(response)
		ctx.Request = httptest.NewRequest(http.MethodGet, "/play/owned/1", nil)
		ctx.Set("auth_required", true)
		ctx.Params = gin.Params{{Key: "hash", Value: source}, {Key: "id", Value: "1"}}
		play(ctx)
		if response.Code != http.StatusBadRequest || !ctx.IsAborted() {
			t.Fatalf("credential-free source was not refused before parsing: %d", response.Code)
		}
	}
	if fetched.Load() != 0 {
		t.Fatal("credential-free playback fetched a remote source")
	}
}

func TestMissingAccountsDoNotDisablePlaybackSourceGate(t *testing.T) {
	previousPath, previousAuth := settings.Path, settings.HttpAuth
	settings.Path, settings.HttpAuth = t.TempDir(), true
	t.Cleanup(func() { settings.Path, settings.HttpAuth = previousPath, previousAuth })
	var fetched atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fetched.Add(1); w.WriteHeader(404) }))
	defer upstream.Close()
	router := gin.New()
	auth.SetupAuth(router)
	router.GET("/stream", stream)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/stream?play&link="+upstream.URL, nil))
	if response.Code != http.StatusBadRequest || fetched.Load() != 0 {
		t.Fatal("missing account file opened unauthenticated source fetch", response.Code)
	}
}
