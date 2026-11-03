package flow

import (
	"bytes"
	"context"
	"errors"
	"golang.org/x/text/encoding/charmap"
	textunicode "golang.org/x/text/encoding/unicode"
	"strings"
	"testing"
)

func TestSubtitleMatchingDoesNotSelectOtherEpisodesOrDirectories(t *testing.T) {
	files := []EpisodeFile{{ID: 1, Path: "Show/Show.S01E01.1080p.mkv", Length: 100}, {ID: 2, Path: "Show/Show.S01E02.mkv", Length: 100}, {ID: 3, Path: "Show/Show.S01E01.1080p.ru.forced.srt", Length: 10}, {ID: 4, Path: "Show/Subs/Show.S01E01.en.ass", Length: 10}, {ID: 5, Path: "Show/Show.S01E02.ru.srt", Length: 10}, {ID: 6, Path: "Elsewhere/Show.S01E01.srt", Length: 10}}
	rows, truncated := MatchSubtitles(files, 1)
	if truncated || len(rows) != 2 {
		t.Fatal(rows, truncated)
	}
	if rows[0].Match != "exact" || rows[0].Language != "ru" || !rows[0].Forced || rows[1].Match != "episode" {
		t.Fatal(rows)
	}
	files = append(files, EpisodeFile{ID: 7, Path: "Show/Show.S01E01.2160p.mkv", Length: 100})
	rows, _ = MatchSubtitles(files, 1)
	if rows[1].Match != "ambiguous" {
		t.Fatal("ambiguous release was guessed", rows)
	}
	for i := 0; i < 105; i++ {
		files = append(files, EpisodeFile{ID: 100 + i, Path: "Show/Show.S01E01.1080p.en.srt", Length: 10})
	}
	rows, truncated = MatchSubtitles(files, 1)
	if len(rows) != 100 || !truncated {
		t.Fatal("candidate bound")
	}
}
func TestSubtitleConversionEscapesPayloadAndPreservesTiming(t *testing.T) {
	srt := []byte("1\r\n00:00:01,125 --> 00:00:02,500\r\n<script>alert(1)</script> & text\r\n")
	out, err := SubtitleWebVTT(srt, ".srt")
	if err != nil || !bytes.Contains(out, []byte("00:00:01.125 --> 00:00:02.500")) || bytes.Contains(out, []byte("<script>")) {
		t.Fatal(string(out), err)
	}
	ass := []byte("[Events]\nFormat: Layer, Start, End, Text\nDialogue: 0,0:00:01.25,0:00:02.50,{\\b1}Hello, world\\Nsecond line\n")
	out, err = SubtitleWebVTT(ass, ".ass")
	if err != nil || !bytes.Contains(out, []byte("Hello, world\nsecond line")) || bytes.Contains(out, []byte(`{\b1}`)) {
		t.Fatal(string(out), err)
	}
	if from, err := subtitleTime("999:59:59.999"); err != nil || from != 3599999999 {
		t.Fatal("32-bit-safe timing", from, err)
	}
}

func TestSubtitleSpecialEpisodesAndLanguageSuffixes(t *testing.T) {
	files := []EpisodeFile{{ID: 1, Path: "Show/Show.S00E01.2160p.mkv", Length: 100}, {ID: 3, Path: "Show/subs/Show.S00E01.[jpn].forced.srt", Length: 10}, {ID: 5, Path: "Show/Show.S00E02.eng.srt", Length: 10}}
	rows, truncated := MatchSubtitles(files, 1)
	if truncated || len(rows) != 1 || rows[0].Language != "ja" || !rows[0].Forced || rows[0].Match != "episode" {
		t.Fatal(rows, truncated)
	}
	if _, _, _, _, ok := episodeIdentity(files[0].Path); ok {
		t.Fatal("subtitle matching expanded automatic warmup eligibility")
	}
}
func TestSubtitleEncodingsAndMalformedLimits(t *testing.T) {
	text := "1\n00:00:01,000 --> 00:00:02,000\nПривет\n"
	for _, encode := range []func([]byte) ([]byte, error){charmap.Windows1251.NewEncoder().Bytes, textunicode.UTF16(textunicode.LittleEndian, textunicode.UseBOM).NewEncoder().Bytes, textunicode.UTF16(textunicode.BigEndian, textunicode.UseBOM).NewEncoder().Bytes, func(data []byte) ([]byte, error) { return append([]byte{0xef, 0xbb, 0xbf}, data...), nil }} {
		input, err := encode([]byte(text))
		if err != nil {
			t.Fatal(err)
		}
		out, err := SubtitleWebVTT(input, ".srt")
		if err != nil || !bytes.Contains(out, []byte("Привет")) {
			t.Fatal(string(out), err)
		}
	}
	for _, input := range [][]byte{[]byte("garbage"), []byte("1\n00:00:03,000 --> 00:00:02,000\nBackwards"), {0, 1, 2}, bytes.Repeat([]byte("a"), SubtitleInputLimit+1)} {
		if _, err := SubtitleWebVTT(input, ".srt"); err == nil {
			t.Fatal("accepted malformed subtitle")
		}
	}
	huge := strings.Repeat("1\n00:00:01,000 --> 00:00:02,000\n"+strings.Repeat("&", 200000)+"\n\n", 10)
	if _, err := SubtitleWebVTT([]byte(huge), ".srt"); err == nil {
		t.Fatal("accepted oversized escaped output")
	}
}

func TestOriginalSubtitleValidationAndCancelledConversion(t *testing.T) {
	input := []byte("1\n00:00:01,000 --> 00:00:02,000\nHello\n")
	if err := ValidateSubtitleContext(context.Background(), input, ".srt"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateSubtitleContext(context.Background(), []byte{0, 1, 2}, ".srt"); err == nil {
		t.Fatal("original binary accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := SubtitleWebVTTContext(ctx, input, ".srt"); !errors.Is(err, context.Canceled) {
		t.Fatal("conversion ignored cancellation", err)
	}
}
