package torr

import (
	"server/lt"
	"testing"
)

func TestImmutableFileIndexPreservesLegacyIDsAndCopies(t *testing.T) {
	files := []*File{{Index: 4, Path: "show/episode2.mkv", Length: 200, Offset: 100}, {Index: 0, Path: "show/episode1.mkv", Length: 100}}
	torrent := &Torrent{}
	// A published immutable metadata snapshot avoids calling a native engine in
	// this test. Production populates it only after the real handle returns files.
	torrent.lh.Store(&lt.Torrent{})
	torrent.filesSnapshot.Store(&torrentFiles{sorted: files, byID: map[int]*File{5: files[0], 1: files[1]}})
	if file := torrent.fileByID(5); file == nil || file.Index != 4 || file.Offset != 100 {
		t.Fatal(file)
	}
	copy := torrent.Files()
	copy[0].Path = "modified"
	file := torrent.fileByID(5)
	file.Length = 0
	if torrent.Files()[0].Path != "show/episode2.mkv" || torrent.fileByID(5).Length != 200 {
		t.Fatal("caller mutated authoritative metadata")
	}
	if torrent.fileByID(2) != nil {
		t.Fatal("sorted position changed the legacy native index")
	}
}
