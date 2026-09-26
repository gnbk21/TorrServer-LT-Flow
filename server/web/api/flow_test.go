package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestFlowControlRejectsUnknownAction(t *testing.T) {
	gin.SetMode(gin.TestMode)
	route := gin.New()
	route.POST("/flow/control", flowControl)
	for _, body := range []string{`{"action":"delete"}`, `{}`, `not-json`} {
		request := httptest.NewRequest(http.MethodPost, "/flow/control", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		route.ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("body %q: status %d", body, response.Code)
		}
	}
}
