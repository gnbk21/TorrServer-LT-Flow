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
		router := gin.New()
		router.Use(PreserveRejectedBody())
		router.POST("/settings", CheckAuth(), func(c *gin.Context) { t.Error("unauthorized handler ran") })
		body := strings.NewReader(`{"action":"get"}`)
		request := httptest.NewRequest("POST", "/settings", body)
		request.Close = true
		request.Header.Set("Accept", accept)
		router.ServeHTTP(writer, request)
		if writer.Code != 401 || body.Len() != 0 {
			t.Fatalf("rejection did not consume the small body: status=%d, remaining=%d", writer.Code, body.Len())
		}
		if (writer.Header().Get("WWW-Authenticate") != "") != (accept == "*/*") {
			t.Fatal("authentication challenge compatibility changed")
		}
	}
}

func TestRejectedBodyMiddlewareEarlyResponses(t *testing.T) {
	for _, status := range []int{403, 429, 503} {
		writer := &rejectedWriter{ResponseRecorder: httptest.NewRecorder()}
		router := gin.New()
		router.Use(PreserveRejectedBody(), func(c *gin.Context) { c.AbortWithStatus(status) })
		router.POST("/settings", func(c *gin.Context) { t.Error("rejected handler ran") })
		body := strings.NewReader(`{"action":"get"}`)
		request := httptest.NewRequest("POST", "/settings", body)
		request.Close = true
		router.ServeHTTP(writer, request)
		if writer.Code != status || body.Len() != 0 {
			t.Fatalf("status %d: code=%d, unread=%d", status, writer.Code, body.Len())
		}
	}
	writer := &rejectedWriter{ResponseRecorder: httptest.NewRecorder()}
	router := gin.New()
	router.Use(PreserveRejectedBody())
	router.POST("/settings", func(c *gin.Context) { c.Status(200) })
	body := strings.NewReader("body")
	request := httptest.NewRequest("POST", "/settings", body)
	request.Close = true
	router.ServeHTTP(writer, request)
	if body.Len() != 4 || len(writer.deadlines) != 0 {
		t.Fatal("middleware altered a successful response")
	}
}
