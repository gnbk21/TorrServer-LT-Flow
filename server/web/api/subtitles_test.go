package api

import (
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"server/flow"
	"server/torr"
	"testing"
)

func TestSubtitleAliasesAreScopedAndHeadDoesNotFetch(t *testing.T) {
	// Public IDs preserve native padding gaps; they are not list ordinals.
	torrent := &torr.Torrent{Data: `{"TorrServer":{"Files":[{"id":1,"path":"Show.S01E01.mkv","length":100},{"id":3,"path":"Show.S01E01.en.srt","length":50},{"id":5,"path":"Show.S01E02.srt","length":50}]}}`}
	for _, row := range []struct {
		resource string
		selected int
		handled  bool
		status   int
	}{{"/media.mkv", 0, false, 200}, {"/media.srt", 0, true, 404}, {"/media.srt", 3, true, 200}, {"/media.en.srt", 3, true, 200}, {"/media.ass", 3, true, 404}, {"/../Show.S01E02.srt", 3, true, 404}, {"/media.srt", 5, true, 404}, {"/media.srt", 2, true, 404}} {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("HEAD", "/flow/play/token"+row.resource, nil)
		c.Params = gin.Params{{Key: "resource", Value: row.resource}}
		handled := capabilityResource(c, torrent, flow.PlaybackClaim{Index: 1, SubtitleIndex: row.selected})
		c.Writer.WriteHeaderNow()
		if handled != row.handled || w.Code != row.status || w.Body.Len() != 0 || torrent.LTHandle() != nil {
			t.Fatal(row, handled, w.Code, w.Body.String())
		}
	}
}
