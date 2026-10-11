package torr

import (
	"net/http"
	"net/http/httptest"
	"server/settings"
	"strings"
	"sync/atomic"
	"testing"
)

func TestPosterSearchReadBoundAndUnavailableInterface(t *testing.T) {
	previous := settings.BTsets()
	settings.StoreBTsets(&settings.BTSets{Flow: settings.DefaultFlowSettings()})
	t.Cleanup(func() { settings.StoreBTsets(previous) })
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		_, _ = w.Write([]byte(`{"results":[]}` + strings.Repeat(" ", 4<<20)))
	}))
	defer srv.Close()
	cfg := settings.TMDBConfig{APIURL: srv.URL}
	if _, err := searchTMDBPoster(cfg, "owned fixture", 0, "en"); err == nil || !strings.Contains(err.Error(), "exceeds 4 MiB") {
		t.Fatalf("response bound not enforced: %v", err)
	}
	flow := settings.DefaultFlowSettings()
	flow.TorrentInterface, flow.RequireTorrentInterface = "Flow-missing-tmdb-interface-418", true
	settings.StoreBTsets(&settings.BTSets{Flow: flow})
	if _, err := searchTMDBPoster(cfg, "owned fixture", 0, "en"); err == nil {
		t.Fatal("unavailable binding bypassed")
	}
	if calls.Load() != 1 {
		t.Fatalf("unbound request reached endpoint: %d", calls.Load())
	}
}

func TestCleanPosterQuery(t *testing.T) {
	cases := []struct {
		in    string
		query string
		year  int
	}{
		{"The.Super.Mario.Galaxy.Movie.2026.1080p.BluRay.x264-BAT1_EniaHD.mkv", "The Super Mario Galaxy Movie", 2026},
		{"Mad.Max.Fury.Road.2015.2160p.UHD.BDRemux.HDR.DV", "Mad Max Fury Road", 2015},
		{"Отражения / Reflections (2023) WEB-DL 1080p", "Отражения", 2023},
		{"rutor.info_Майкл / Michael (2026) WEB-DL 1080p | D | UltradoxStudio", "Майкл", 2026},
		{"rutor.info_Майкл_/_Michael_(2026)_WEB-DL_1080p_|_D_|_UltradoxStudio", "Майкл", 2026},
		{"What.Is.Love.2010.1080p.BluRay", "What Is Love", 2010},
		{"Дюна: Часть вторая / Dune: Part Two [2024, BDRip]", "Дюна: Часть вторая", 2024},
		{"Breaking.Bad.S05E14.720p.HDTV.x264", "Breaking Bad", 0},
		{"Interstellar (2014) BDRip 1080p", "Interstellar", 2014},
		{"Some Movie 1080p WEBRip", "Some Movie", 0},
		{"simple name", "simple name", 0},
		{"", "", 0},
		{"1080p x264", "", 0},
		{"38cef0b40f595e22f47e33de0acfef44a554c3b8", "", 0},
	}
	for _, c := range cases {
		q, y := CleanPosterQuery(c.in)
		if q != c.query || y != c.year {
			t.Errorf("CleanPosterQuery(%q) = (%q, %d), want (%q, %d)", c.in, q, y, c.query, c.year)
		}
	}
}

func TestHasCyrillic(t *testing.T) {
	if hasCyrillic("Dune Part Two") {
		t.Error("latin-only string reported as cyrillic")
	}
	if !hasCyrillic("Дюна") {
		t.Error("cyrillic string not detected")
	}
}
