package flow

import "testing"

func TestNextEpisodeConfidence(t *testing.T) {
	files := []EpisodeFile{{ID: 1, Path: "Show/Show.S01E01.mkv", Length: 100}, {ID: 2, Path: "Show/Show.S01E02.mkv", Length: 100}}
	if next, ok := ConfidentNextEpisode(files, 1); !ok || next.ID != 2 {
		t.Fatal(next, ok)
	}
	for _, name := range []string{"Show/Show.S01E02.mp4", "Show/Show.S01E01E02.mkv", "Other/Show.S01E02.mkv", "Show/Show.S02E01.mkv", "Show/Other.S01E02.mkv", "Show/Show.S01E02.sample.mkv"} {
		copy := append([]EpisodeFile(nil), files...)
		if name == "Show/Show.S01E02.mp4" {
			copy = append(copy, EpisodeFile{ID: 3, Path: name, Length: 100})
		} else {
			copy[1].Path = name
		}
		if next, ok := ConfidentNextEpisode(copy, 1); ok {
			t.Fatalf("ambiguous %q: %+v", name, next)
		}
	}
	files[0].Path, files[1].Path = "Show/Show.1x01.mkv", "Show/Show.1x02.mkv"
	if _, ok := ConfidentNextEpisode(files, 1); !ok {
		t.Fatal("NxNN rejected")
	}
}

func TestWarmupWholePieceBudget(t *testing.T) {
	for _, plen := range []int64{16384, 1 << 20, 4 << 20, 16 << 20, 32 << 20, 64 << 20} {
		for _, length := range []int64{1, plen, 3 * plen, 100 * plen} {
			pieces, bytes := WarmupPieces(plen-1, length, plen, 1024)
			if bytes > NextEpisodeMaxBytes || bytes != int64(len(pieces))*plen || len(pieces) > 2048 {
				t.Fatal(plen, length, bytes)
			}
			seen := map[int]bool{}
			for _, p := range pieces {
				if seen[p] || p < 0 || p >= 1024 {
					t.Fatal(pieces)
				}
				seen[p] = true
			}
			if plen <= NextEpisodeMaxBytes && (len(pieces) == 0 || pieces[0] != 0) {
				t.Fatal("missing head", plen, length, pieces)
			}
		}
	}
	for _, args := range [][4]int64{{-1, 1, 1, 2}, {0, 0, 1, 2}, {0, 1, 0, 2}, {10, 100, 4, 2}, {1, 1<<63 - 1, 1, 10}} {
		if p, _ := WarmupPieces(args[0], args[1], args[2], int(args[3])); len(p) != 0 {
			t.Fatal(args, p)
		}
	}
}
