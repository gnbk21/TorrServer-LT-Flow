package api

import (
	"encoding/json"
	"server/torr/state"
	"server/torrshash"
	"strings"
	"testing"
)

func TestLibraryExportUsesAllowlistAndRebuildsTokens(t *testing.T) {
	rows := []state.TorrentStatus{{Hash: strings.Repeat("AB", 20), Title: "Episode https://user:secret@tracker.test/private-secret?passkey=secret", Category: "series",
		Data: `{"password":"secret"}`, Poster: "https://images.test/?token=secret", TorrsHash: "opaque-private-secret", FileStats: []*state.TorrentFileStat{{Path: "C:/private-secret"}}},
		{Hash: strings.Repeat("cd", 32), Title: "v2 episode"}}
	exported, err := projectLibraryExport(rows)
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(exported)
	for _, forbidden := range []string{"secret", "password", "poster", "data", "file_stats"} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("export leaked %s", forbidden)
		}
	}
	token, err := torrshash.Unpack(exported[0].TorrsHash)
	if err != nil || token.Hash != strings.ToLower(rows[0].Hash) || token.Title() != exported[0].Title || token.Category() != "series" || len(token.Trackers()) != 0 || token.Poster() != "" {
		t.Fatalf("unsafe or incompatible compact export: %v", err)
	}
	if exported[1].Hash != rows[1].Hash || exported[1].TorrsHash != "" {
		t.Fatal("v2 identity truncated or forced into v1 compact format")
	}
	if rows[0].TorrsHash != "opaque-private-secret" || !strings.Contains(rows[0].Title, "secret") {
		t.Fatal("export mutated runtime data")
	}
}

func TestLibraryExportRejectsPartialOrOversizedOutput(t *testing.T) {
	for _, rows := range [][]state.TorrentStatus{
		{{Hash: "invalid"}},
		{{Hash: strings.Repeat("a", 40), Title: strings.Repeat("x", 4097)}},
		make([]state.TorrentStatus, libraryExportRows+1),
	} {
		if _, err := projectLibraryExport(rows); err == nil {
			t.Fatal("invalid export was silently accepted")
		}
	}
	rows := make([]state.TorrentStatus, libraryExportRows)
	for i := range rows {
		rows[i] = state.TorrentStatus{Hash: strings.Repeat("a", 64), Title: strings.Repeat("x", 4096)}
	}
	if _, err := projectLibraryExport(rows); err == nil {
		t.Fatal("export exceeded total byte budget")
	}
	empty, err := projectLibraryExport(nil)
	if err != nil || empty == nil || len(empty) != 0 {
		t.Fatal("empty library must export as an empty array")
	}
}
