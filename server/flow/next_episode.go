package flow

import (
	"path"
	"regexp"
	"strconv"
	"strings"
)

const NextEpisodeMaxBytes int64 = 32 << 20

type EpisodeFile struct {
	ID             int
	Path           string
	Offset, Length int64
}

var nextEpisodeCode = regexp.MustCompile(`(?i)(?:^|[^a-z0-9])(?:s([0-9]{1,3})[._ -]*e([0-9]{1,4})|([0-9]{1,2})x([0-9]{1,3}))(?:[^a-z0-9]|$)`)
var extraEpisodeCode = regexp.MustCompile(`(?i)(?:e[0-9]+|[0-9]+x[0-9]+)`)
var excludedEpisode = regexp.MustCompile(`(?i)(?:^|[._ -])(?:sample|trailer|preview)(?:[._ -]|$)`)

func episodeIdentity(name string) (directory, prefix string, season, episode int, ok bool) {
	name = strings.ReplaceAll(name, "\\", "/")
	base := path.Base(name)
	if excludedEpisode.MatchString(base) {
		return
	}
	ext := strings.ToLower(path.Ext(base))
	if ext != ".mkv" && ext != ".mp4" && ext != ".avi" && ext != ".ts" && ext != ".m4v" && ext != ".webm" {
		return
	}
	match := nextEpisodeCode.FindAllStringSubmatchIndex(base, -1)
	if len(match) != 1 {
		return
	}
	m := match[0]
	// A second episode token (S01E01E02 or S01E01-E02) is a bundle, not an
	// unambiguous single episode. Never silently choose a release from it.
	if extraEpisodeCode.MatchString(base[m[1]:]) {
		return
	}
	read := func(a, b int) int {
		if a < 0 {
			return 0
		}
		n, _ := strconv.Atoi(base[a:b])
		return n
	}
	season, episode = read(m[2], m[3]), read(m[4], m[5])
	if m[2] < 0 {
		season, episode = read(m[6], m[7]), read(m[8], m[9])
	}
	if season < 1 || episode < 1 {
		return
	}
	directory = strings.ToLower(path.Dir(name))
	prefix = strings.ToLower(strings.Trim(base[:m[0]], "._ -"))
	ok = true
	return
}

// ConfidentNextEpisode deliberately accepts fewer patterns than the playlist
// parser. Guessing a season change, edition or matching title would fetch the
// wrong file. Ambiguous releases remain available through explicit selection.
func ConfidentNextEpisode(files []EpisodeFile, current int) (EpisodeFile, bool) {
	var source EpisodeFile
	for _, f := range files {
		if f.ID == current {
			source = f
			break
		}
	}
	dir, prefix, season, episode, ok := episodeIdentity(source.Path)
	if !ok {
		return EpisodeFile{}, false
	}
	var result EpisodeFile
	for _, f := range files {
		d, p, s, e, parsed := episodeIdentity(f.Path)
		if !parsed || d != dir || p != prefix || s != season || e != episode+1 || f.Length <= 0 {
			continue
		}
		if result.ID != 0 {
			return EpisodeFile{}, false
		}
		result = f
	}
	return result, result.ID != 0
}

// WarmupPieces charges whole boundary pieces, never just the requested file
// bytes. Tiny pieces are capped at 2048 entries; no unbounded priority list.
func WarmupPieces(offset, length, pieceLength int64, count int) ([]int, int64) {
	if offset < 0 || length <= 0 || pieceLength <= 0 || pieceLength > NextEpisodeMaxBytes || count <= 0 {
		return nil, 0
	}
	if length > int64(^uint64(0)>>1)-offset {
		return nil, 0
	}
	first, last := offset/pieceLength, (offset+length-1)/pieceLength
	if first >= int64(count) || last >= int64(count) {
		return nil, 0
	}
	limit := min(int64(2048), NextEpisodeMaxBytes/pieceLength)
	// Reserve up to a quarter of the fixed cap for an EOF container index.
	tail := min(last-first+1, max(int64(1), limit/4))
	head := min(last-first+1-tail, limit-tail)
	if head == 0 {
		head, tail = 1, max(int64(0), tail-1)
	}
	pieces := make([]int, 0, head+tail)
	for i := first; i < first+head; i++ {
		pieces = append(pieces, int(i))
	}
	for i := max(first+head, last-tail+1); i <= last; i++ {
		pieces = append(pieces, int(i))
	}
	return pieces, int64(len(pieces)) * pieceLength
}
