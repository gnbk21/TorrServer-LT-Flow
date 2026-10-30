package api

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	sets "server/settings"
	"server/torr/state"
)

func TestFindFileNamesakesMatchesOnlyRelatedTracks(t *testing.T) {
	paths := []string{
		"Show/Show S01E1.mkv",
		"Show/Show S01E10.mkv",
		"Show/Show S01E10.rus.mka",
		"Show/Show S01E11.srt",
		"Show/Show S01E1.rus.mka",
		"Show/Sound/Show S01E1 [Studio].ac3",
		"Show/Subs/Show S01E1/English.srt",
		"Show/Show S01E1.FORCED.ASS",
		"Show/Show S01E1.jpg",
		"Show/Show S01E1.nfo",
		"Show/Show S01E1.m3u",
		"Show/Show S01E1 sample.mkv",
	}
	files := make([]*state.TorrentFileStat, len(paths))
	for i, path := range paths {
		files[i] = &state.TorrentFileStat{Id: i + 1, Path: path}
	}
	var got []string
	for _, f := range findFileNamesakes(files, files[0]) {
		got = append(got, f.Path)
	}
	if want := paths[4:8]; !slices.Equal(got, want) {
		t.Fatalf("tracks = %q, want %q", got, want)
	}
}

func TestContainsMediaNameWordBoundaries(t *testing.T) {
	for _, tc := range []struct {
		path, name string
		want       bool
	}{
		{"Show S01E1.rus.mka", "Show S01E1", true},
		{"Show S01E10.rus.mka", "Show S01E1", false},
		{"Subs/Show - 1/English.srt", "Show - 1", true},
		{"Subs/Show - 10/English.srt", "Show - 1", false},
		{"Alien.eng.srt", "Alien", true},
		{"Aliens.eng.srt", "Alien", false},
		{"NotAlien.eng.srt", "Alien", false},
		{"Aliens/Alien.eng.srt", "Alien", true},
		{"Фильм.ru.srt", "Фильм", true},
		{"Фильмы.ru.srt", "Фильм", false},
		{"СуперФильм.ru.srt", "Фильм", false},
		{"Show 1я.srt", "Show 1", false},
		{"Show 1٢.srt", "Show 1", false},
		{"Cafe\u0301.eng.srt", "Cafe", false},
		{"[Movie].eng.srt", "[Movie]", true},
		{"anything.srt", "", false},
		{"", "Movie", false},
	} {
		t.Run(tc.path+"/"+tc.name, func(t *testing.T) {
			if got := containsMediaName(tc.path, tc.name); got != tc.want {
				t.Fatalf("containsMediaName(%q, %q) = %v", tc.path, tc.name, got)
			}
		})
	}
}

func TestSendM3UETagTracksBodyAndClientSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	send := func(name, hash, body, oldTag string) *httptest.ResponseRecorder {
		r := gin.New()
		r.GET("/playlist", func(c *gin.Context) { sendM3U(c, name, hash, body) })
		req := httptest.NewRequest(http.MethodGet, "/playlist", nil)
		if oldTag != "" {
			req.Header.Set("If-None-Match", oldTag)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}
	first := "#EXTM3U\nhttp://pc/stream/movie?index=1&ss=first\n"
	response := send("movie.m3u", "hash", first, "")
	tag := response.Header().Get("ETag")
	if response.Code != 200 || tag == "" || response.Body.String() != first {
		t.Fatal(response.Code, response.Header(), response.Body.String())
	}
	if w := send("movie.m3u", "hash", first, tag); w.Code != 304 || w.Body.Len() != 0 {
		t.Fatalf("unchanged body: %d %q", w.Code, w.Body.String())
	}
	for _, body := range []string{
		strings.ReplaceAll(first, "ss=first", "ss=second"),
		strings.ReplaceAll(first, "index=1", "index=2"),
		strings.ReplaceAll(first, "http://pc", "https://pc"),
		"#EXTM3U\n#EXTINF:0,Renamed episode\n" + first,
	} {
		w := send("movie.m3u", "hash", body, tag)
		if w.Code != 200 || w.Header().Get("ETag") == tag || w.Body.String() != body {
			t.Fatalf("changed body: %d %q %q", w.Code, w.Header().Get("ETag"), w.Body.String())
		}
	}
	if w := send("movie.m3u", "", first, ""); w.Header().Get("ETag") != "" {
		t.Fatal("hashless playlist acquired an ETag")
	}
}

func TestPlaylistFetchesKeepIndependentSessions(t *testing.T) {
	old := sets.BTsets()
	sets.StoreBTsets(sets.NewDefaultConfig())
	t.Cleanup(func() { sets.StoreBTsets(old) })
	tor := &state.TorrentStatus{
		Hash: strings.Repeat("a", 40),
		FileStats: []*state.TorrentFileStat{
			{Id: 1, Path: "Show/Show S01E1.mkv"},
			{Id: 2, Path: "Show/Show S01E1.rus.mka"},
			{Id: 3, Path: "Show/Show S01E2.mkv"},
		},
	}
	session := func(body string) string {
		t.Helper()
		found := ""
		for _, line := range strings.Split(body, "\n") {
			// VLC's input-slave URLs are #-separated, whereas each main
			// video occupies its own line. Both must use this fetch's token.
			line = strings.TrimPrefix(line, "#EXTVLCOPT:input-slave=")
			for _, raw := range strings.Split(line, "#") {
				if !strings.HasPrefix(raw, "http://") {
					continue
				}
				u, err := url.Parse(raw)
				if err != nil {
					t.Fatal(err)
				}
				value := u.Query().Get("ss")
				if value == "" || (found != "" && value != found) {
					t.Fatalf("missing or inconsistent session in %q", body)
				}
				found = value
			}
		}
		if found == "" {
			t.Fatal("no direct stream URLs")
		}
		return found
	}
	a := session(getM3uList(tor, "http://pc:8090", false, ""))
	b := session(getM3uList(tor, "http://pc:8090", false, ""))
	if a == b {
		t.Fatal("separate playlist fetches share a playback session")
	}
}
