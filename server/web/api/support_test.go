package api

import (
	"net/http"
	"net/http/httptest"
	sets "server/settings"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSupportRedactsNewHostPolicyWithoutMutatingSettings(t *testing.T) {
	old := sets.BTsets()
	t.Cleanup(func() { sets.StoreBTsets(old) })
	f := sets.DefaultFlowSettings()
	f.ManagementOrigins = "https://private-controller.example"
	f.TorrentInterface = "Private adapter name"
	sets.StoreBTsets(&sets.BTSets{Flow: f})
	r := gin.New()
	r.GET("/support", supportReport)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/support", nil))
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), f.ManagementOrigins) || strings.Contains(w.Body.String(), f.TorrentInterface) {
		t.Fatal("support export disclosed host policy")
	}
	if sets.CurrentFlow().TorrentInterface != f.TorrentInterface || sets.CurrentFlow().ManagementOrigins != f.ManagementOrigins {
		t.Fatal("support export changed live policy")
	}
}
