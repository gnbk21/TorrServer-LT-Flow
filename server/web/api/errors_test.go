package api

import (
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestVersionedFlowErrorsKeepLegacyFieldsAndHeadSemantics(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodHead} {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(method, "/flow/recent", nil)
		abortWithErrorCode(c, 503, "SERVICE_UNAVAILABLE", errors.New("existing error text"), 1)
		if w.Code != 503 || w.Header().Get("Cache-Control") != "private, no-store" {
			t.Fatal(w.Code)
		}
		if method == http.MethodHead {
			if w.Body.Len() != 0 {
				t.Fatal("HEAD error returned a body")
			}
			continue
		}
		var body struct {
			SchemaVersion int `json:"schema_version"`
			Error, Code   string
		}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body.SchemaVersion != 1 || body.Error != "existing error text" || body.Code != "SERVICE_UNAVAILABLE" {
			t.Fatal(body)
		}
	}
}

func TestStructuredErrorsKeepLegacyTextAndNeverInferStarting(t *testing.T) {
	for _, status := range []int{401, 403, 429, 503} {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		abortWithJSONError(c, status, errors.New("existing error text"))
		var body map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if w.Code != status || body["error"] != "existing error text" || body["code"] != httpErrorCode(status) || body["code"] == "ENGINE_STARTING" {
			t.Fatal(body)
		}
	}
}

func TestErrorMessageRemovesCredentialsAndBoundsUntrustedDetails(t *testing.T) {
	message := errorMessage(errors.New(`request https://user:password@example.test/private-passkey/announce?token=credential failed; Authorization: Bearer topsecret`))
	for _, secret := range []string{"user:", "password", "private-passkey", "credential", "topsecret"} {
		if strings.Contains(message, secret) {
			t.Fatal("credential exposed in API error")
		}
	}
	message = errorMessage(errors.New(strings.Repeat("例", 5000)))
	if len(message) > 4100 || !utf8.ValidString(message) {
		t.Fatal("error response is unbounded or invalid UTF-8")
	}
	if errorMessage(errors.New("metadata timed out")) != "metadata timed out" {
		t.Fatal("useful explanation removed")
	}
}
