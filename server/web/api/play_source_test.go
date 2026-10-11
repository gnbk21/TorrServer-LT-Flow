package api

import (
	"net/http"
	"net/http/httptest"
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
