package auth

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"server/settings"
)

func TestCheckAuthConsumesSmallRejectedBody(t *testing.T) {
	previous := settings.HttpAuth
	settings.HttpAuth = true
	t.Cleanup(func() { settings.HttpAuth = previous })
	for _, accept := range []string{"*/*", "application/json"} {
		writer := &rejectedWriter{ResponseRecorder: httptest.NewRecorder()}
		context, _ := gin.CreateTestContext(writer)
		body := strings.NewReader(`{"action":"get"}`)
		context.Request = httptest.NewRequest("POST", "/settings", body)
		context.Request.Close = true
		context.Request.Header.Set("Accept", accept)
		CheckAuth()(context)
		if writer.Code != 401 || !context.IsAborted() || body.Len() != 0 {
			t.Fatalf("rejection did not consume the small body: status=%d, remaining=%d", writer.Code, body.Len())
		}
		if (writer.Header().Get("WWW-Authenticate") != "") != (accept == "*/*") {
			t.Fatal("authentication challenge compatibility changed")
		}
	}
}
