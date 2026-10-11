package api

import (
	"fmt"
	"server/torr/state"
	"testing"
)

func TestLibraryFiltersEntireDatasetBeforePagination(t *testing.T) {
	rows := make([]state.TorrentStatus, 5000)
	for i := range rows {
		rows[i] = state.TorrentStatus{Hash: fmt.Sprintf("%040x", i), Title: fmt.Sprintf("Episode %04d", i), Category: "series", Timestamp: int64(i), TorrentSize: int64(5000 - i)}
	}
	rows[4999].Category = "other"
	result := projectLibrary(rows, "Episode 49", "category:series", "recent", 2, 50)
	if result.Total != 99 || result.LibraryTotal != 5000 || result.Page != 2 || len(result.Items) != 49 || result.Items[0].Timestamp != 4948 {
		t.Fatalf("incorrect full-set projection: %+v", result)
	}
	if len(result.Categories) != 2 {
		t.Fatal("categories incorrectly derived from a page/filter")
	}
	last := projectLibrary(rows, "missing", "", "size", 1000000, 50)
	if last.Page != 1 || len(last.Items) != 0 {
		t.Fatal("empty/clamped page invalid")
	}
}

func TestLibraryStableTieOrderAndUncategorized(t *testing.T) {
	rows := []state.TorrentStatus{{Hash: "b", Title: "SAME"}, {Hash: "a", Title: "same"}, {Hash: "c", Title: "same", Category: "films"}}
	result := projectLibrary(rows, "same", "uncategorized", "title", 1, 1)
	if result.Total != 2 || result.Items[0].Hash != "a" {
		t.Fatalf("unstable tie ordering: %+v", result)
	}
}
