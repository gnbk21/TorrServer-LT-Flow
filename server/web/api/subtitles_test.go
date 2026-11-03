package api

import (
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"server/flow"
	"server/torr"
	"testing"
	"time"
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

func TestSubtitleGlobalAdmissionKeepsHeadMetadataOnly(t *testing.T) {
	torrent := &torr.Torrent{Data: `{"TorrServer":{"Files":[{"id":1,"path":"video.mkv","length":100},{"id":3,"path":"video.en.srt","length":50}]}}`}
	subtitleResponses <- struct{}{}
	subtitleResponses <- struct{}{}
	defer func() { <-subtitleResponses; <-subtitleResponses }()
	for _, method := range []string{"HEAD", "GET"} {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(method, "/subs/fixture/1?track=3", nil)
		serveSelectedSubtitle(c, torrent, 1, 3, "original")
		c.Writer.WriteHeaderNow()
		if method == "HEAD" && (w.Code != 200 || w.Body.Len() != 0) {
			t.Fatal("metadata HEAD consumed read admission", w.Code)
		}
		if method == "GET" && (w.Code != 503 || w.Header().Get("Retry-After") != "1") {
			t.Fatal("global fetch admission bypassed", w.Code)
		}
		if torrent.LTHandle() != nil {
			t.Fatal("admission or HEAD activated torrent")
		}
	}
}

type subtitleDeadlineWriter struct {
	*httptest.ResponseRecorder
	deadlines []time.Time
}

func (w *subtitleDeadlineWriter) SetWriteDeadline(at time.Time) error {
	w.deadlines = append(w.deadlines, at)
	return nil
}

func TestSubtitleDeliveryDeadlineDoesNotLeakToKeepalive(t *testing.T) {
	w := &subtitleDeadlineWriter{ResponseRecorder: httptest.NewRecorder()}
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/subs/fixture/1?track=3", nil)
	torrent := &torr.Torrent{Data: `{"TorrServer":{"Files":[{"id":1,"path":"video.mkv","length":100},{"id":3,"path":"video.en.srt","length":50}]}}`}
	start := time.Now()
	serveSelectedSubtitle(c, torrent, 1, 3, "original")
	if len(w.deadlines) != 2 || w.deadlines[0].Before(start.Add(9*time.Second)) || w.deadlines[0].After(start.Add(11*time.Second)) || !w.deadlines[1].IsZero() {
		t.Fatal("subtitle deadline not bounded/reset", w.deadlines)
	}
	if len(subtitleResponses) != 0 {
		t.Fatal("error retained global response slot")
	}
}
