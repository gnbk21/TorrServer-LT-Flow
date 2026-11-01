package api

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"server/diagnostics"
	sets "server/settings"
	"server/web/auth"
	"strings"
	"testing"
)

func TestPairingProofGrantsNoManagementAuthority(t *testing.T) {
	old, oldAuth := sets.BTsets(), sets.HttpAuth
	t.Cleanup(func() { sets.StoreBTsets(old); sets.HttpAuth = oldAuth })
	sets.StoreBTsets(sets.NewDefaultConfig())
	sets.HttpAuth = true
	r := gin.New()
	r.Use(ManagementPolicy(), auth.BasicAuth(gin.Accounts{"owner": "fixture"}))
	a := r.Group("/", auth.CheckAuth())
	a.POST("/flow/pairing", pairingCreate)
	a.GET("/flow/pairing/status/:id", pairingStatus)
	a.GET("/settings/metadata/v1", settingsMetadata)
	a.POST("/settings", func(c *gin.Context) { c.Status(204) })
	r.GET("/flow/pairing/proof/:token", pairingVisit)
	call := func(method, path, credential string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, "http://127.0.0.1:8090"+path, nil)
		req.RemoteAddr = "192.0.2.10:12345"
		if credential == "owner" {
			req.SetBasicAuth("owner", "fixture")
		} else if credential != "" {
			req.Header.Set("Authorization", "Bearer "+credential)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}
	if call("POST", "/flow/pairing", "").Code != 401 {
		t.Fatal("unauthenticated creation")
	}
	response := call("POST", "/flow/pairing", "owner")
	var created struct {
		Proof diagnostics.PairingProof `json:"proof"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &created); err != nil || response.Code != 201 {
		t.Fatal(response.Code, err)
	}
	if call("GET", "/flow/pairing/proof/"+created.Proof.Token, "").Code != 200 {
		t.Fatal("read-only proof requires auth")
	}
	for _, path := range []string{"/flow/pairing/status/" + created.Proof.ID, "/settings/metadata/v1"} {
		if call("GET", path, created.Proof.Token).Code != 401 {
			t.Fatal("proof became a management credential")
		}
		if call("GET", path, "owner").Code != 200 {
			t.Fatal("authenticated read failed")
		}
	}
	if call("POST", "/settings", created.Proof.Token).Code != 401 {
		t.Fatal("proof changed settings")
	}
	status := call("GET", "/flow/pairing/status/"+created.Proof.ID, "owner")
	if strings.Contains(status.Body.String(), created.Proof.Token) {
		t.Fatal("status leaked proof token")
	}
}
