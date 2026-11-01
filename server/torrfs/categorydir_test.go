package torrfs

import (
	"slices"
	"strings"
	"testing"

	"server/torr"
)

func namedTorrent(title, hash string) *torr.Torrent {
	return &torr.Torrent{Title: title, TorrentSpec: &torr.TorrentSpec{InfoHash: torr.MustParseHash(hash)}}
}

func TestTorrentNamesDisambiguateTitlesAndSanitization(t *testing.T) {
	torrs := []*torr.Torrent{
		namedTorrent("Same", strings.Repeat("11", 20)),
		namedTorrent("Other", strings.Repeat("22", 20)),
		namedTorrent("Same", strings.Repeat("33", 32)),
		namedTorrent("a/b", strings.Repeat("44", 20)),
		namedTorrent("a_b", strings.Repeat("55", 20)),
		namedTorrent("", strings.Repeat("66", 32)),
	}
	want := []string{"Same [11111111]", "Other", "Same [33333333]", "a_b [44444444]", "a_b [55555555]", strings.Repeat("66", 32)}
	if got := torrentNames(torrs); !slices.Equal(got, want) {
		t.Fatalf("names = %q, want %q", got, want)
	}
}

func TestTorrentNamesResolvePrefixAndLiteralCollisions(t *testing.T) {
	h1 := "12345678aaaa" + strings.Repeat("1", 28)
	h2 := "12345678bbbb" + strings.Repeat("2", 28)
	h3 := strings.Repeat("c", 64)
	h4 := strings.Repeat("d", 40)
	torrs := []*torr.Torrent{
		namedTorrent("Same", h1), namedTorrent("Same", h2),
		namedTorrent("Same [12345678aaaa]", h3),
		namedTorrent("Same ["+h1+"]", h4),
	}
	names := torrentNames(torrs)
	if names[0] != "Same ["+h1[:16]+"]" || names[1] != "Same [12345678bbbb]" {
		t.Fatal(names)
	}
	if names[2] != torrs[2].Title || names[3] != torrs[3].Title {
		t.Fatal("unique literal titles were renamed", names)
	}
	byHash := make(map[string]string)
	seen := make(map[string]bool)
	for i, name := range names {
		if seen[name] {
			t.Fatal("duplicate directory", name)
		}
		seen[name], byHash[torrs[i].Hash().HexString()] = true, name
	}
	slices.Reverse(torrs)
	for i, name := range torrentNames(torrs) {
		if name != byHash[torrs[i].Hash().HexString()] {
			t.Fatal("directory depends on list ordering", name)
		}
	}
}

func TestTorrentNamesFullHashLiteralCollision(t *testing.T) {
	h := strings.Repeat("ab", 20)
	torrs := []*torr.Torrent{namedTorrent("Movie", h), namedTorrent("Movie", strings.Repeat("cd", 32))}
	for width := 8; width <= len(h); width += 4 {
		identity := strings.Repeat("ef", 19) + string("0123456789abcdef"[width/4]) + "0"
		torrs = append(torrs, namedTorrent("Movie ["+h[:width]+"]", identity))
	}
	torrs = append(torrs, namedTorrent("Movie ["+h+"] (2)", strings.Repeat("12", 32)))
	names := torrentNames(torrs)
	if names[0] != "Movie ["+h+"] (3)" {
		t.Fatal(names)
	}
	seen := make(map[string]bool)
	for _, name := range names {
		if seen[name] {
			t.Fatal("duplicate directory", name)
		}
		seen[name] = true
	}
}

func TestTorrentNamesV1AndV2WithSamePrefix(t *testing.T) {
	h1 := strings.Repeat("ab", 20)
	h2 := h1 + strings.Repeat("cd", 12)
	torrs := []*torr.Torrent{namedTorrent("Movie", h1), namedTorrent("Movie", h2)}
	names := torrentNames(torrs)
	if names[0] != "Movie ["+h1+"]" || names[1] != "Movie ["+h2[:44]+"]" || names[0] == names[1] {
		t.Fatal(names)
	}
	if got := torrentNames(nil); len(got) != 0 {
		t.Fatal(got)
	}
}
