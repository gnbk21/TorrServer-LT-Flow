package api

import (
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	sets "server/settings"
	"strings"
	"testing"
)

func TestStorageRequestRejectsMalformedAndIneffectivePayloads(t *testing.T) {
	old := sets.ReadOnly
	sets.ReadOnly = false
	defer func() { sets.ReadOnly = old }()
	router := gin.New()
	router.POST("/storage/settings", UpdateStorageSettings)
	for _, body := range []string{`{}`, `null`, `{"settings":true}`, `{"viewed":null}`, `{"settings":""}`, `{"settings":"sqlite"}`, `{"unknown":1}`, `{"viewedCount":12}`, `{"settings":"json","revision":12}`, `{"settings":"json"} {}`} {
		request := httptest.NewRequest("POST", "/storage/settings", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json; charset=utf-8")
		result := httptest.NewRecorder()
		router.ServeHTTP(result, request)
		if result.Code != 400 {
			t.Fatalf("accepted %s: %d", body, result.Code)
		}
	}
	request := httptest.NewRequest("POST", "/storage/settings", strings.NewReader("settings=json&settings=bbolt"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
	result := httptest.NewRecorder()
	router.ServeHTTP(result, request)
	if result.Code != 400 {
		t.Fatal("duplicate form values accepted")
	}
}
