package flow

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"golang.org/x/text/encoding/charmap"
	textunicode "golang.org/x/text/encoding/unicode"
	"golang.org/x/text/unicode/norm"
	"html"
	"path"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

const SubtitleInputLimit = 4 << 20
const SubtitleOutputLimit = 8 << 20

var ErrSubtitle = errors.New("invalid or unsupported subtitle")

type SubtitleCandidate struct {
	FileIndex       int    `json:"file_index"`
	Path            string `json:"path"`
	Size            int64  `json:"size"`
	Format          string `json:"format"`
	Language        string `json:"language,omitempty"`
	Forced          bool   `json:"forced"`
	HearingImpaired bool   `json:"hearing_impaired"`
	Match           string `json:"match"`
}

var subtitleLanguages = map[string]string{
	"ru": "ru", "rus": "ru", "russian": "ru", "en": "en", "eng": "en", "english": "en", "uk": "uk", "ua": "uk", "ukr": "uk", "fr": "fr", "fre": "fr", "fra": "fr", "french": "fr", "ro": "ro", "ron": "ro", "rum": "ro", "romanian": "ro", "zh": "zh", "chi": "zh", "zho": "zh", "chinese": "zh", "bg": "bg", "bul": "bg", "bulgarian": "bg",
	"de": "de", "deu": "de", "ger": "de", "german": "de", "es": "es", "spa": "es", "spanish": "es", "it": "it", "ita": "it", "italian": "it", "pt": "pt", "por": "pt", "portuguese": "pt", "ja": "ja", "jpn": "ja", "japanese": "ja", "ko": "ko", "kor": "ko", "korean": "ko", "ar": "ar", "ara": "ar", "arabic": "ar", "kk": "kk", "kaz": "kk", "kazakh": "kk", "tr": "tr", "tur": "tr", "turkish": "tr", "pl": "pl", "pol": "pl", "polish": "pl", "nl": "nl", "dut": "nl", "nld": "nl", "cs": "cs", "cze": "cs", "ces": "cs", "sv": "sv", "swe": "sv", "fi": "fi", "fin": "fi", "da": "da", "dan": "da", "no": "no", "nor": "no", "nb": "nb", "nob": "nb", "el": "el", "gre": "el", "ell": "el", "he": "he", "heb": "he", "hu": "hu", "hun": "hu", "hin": "hi", "hindi": "hi", "th": "th", "tha": "th", "vi": "vi", "vie": "vi", "id": "id", "ind": "id",
}

func normalizedSubtitlePath(value string) string {
	return strings.ToLower(norm.NFKC.String(strings.ReplaceAll(value, "\\", "/")))
}
func SubtitleExtension(value string) string {
	ext := strings.ToLower(path.Ext(value))
	switch ext {
	case ".srt", ".ass", ".ssa", ".vtt":
		return ext
	}
	return ""
}
func MatchSubtitles(files []EpisodeFile, videoID int) ([]SubtitleCandidate, bool) {
	out := []SubtitleCandidate{}
	truncated := false
	var video EpisodeFile
	for _, f := range files {
		if f.ID == videoID {
			video = f
			break
		}
	}
	if video.ID == 0 || SubtitleExtension(video.Path) != "" {
		return out, false
	}
	videoPath := normalizedSubtitlePath(video.Path)
	dir := path.Dir(videoPath)
	stem := strings.TrimSuffix(path.Base(videoPath), path.Ext(videoPath))
	for _, f := range files {
		ext := SubtitleExtension(f.Path)
		if ext == "" || f.Length <= 0 || f.ID == videoID {
			continue
		}
		if len(f.Path) > 8192 {
			truncated = true
			continue
		}
		value := normalizedSubtitlePath(f.Path)
		subdir := path.Dir(value)
		if subdir != dir && !(path.Dir(subdir) == dir && (path.Base(subdir) == "subs" || path.Base(subdir) == "subtitles")) {
			continue
		}
		base := strings.TrimSuffix(path.Base(value), ext)
		row := SubtitleCandidate{FileIndex: f.ID, Path: f.Path, Size: f.Length, Format: ext[1:]}
		tokens := strings.FieldsFunc(base, func(r rune) bool { return r == '.' || r == '_' || r == ' ' || r == '-' })
		for len(tokens) > 0 {
			last := strings.Trim(tokens[len(tokens)-1], "[]()")
			if lang := subtitleLanguages[last]; lang != "" {
				if row.Language == "" {
					row.Language = lang
				}
			} else if last == "forced" {
				row.Forced = true
			} else if last == "sdh" || last == "cc" || last == "hi" {
				row.HearingImpaired = true
			} else {
				break
			}
			tokens = tokens[:len(tokens)-1]
		}
		// Strip only recognised suffix tokens; punctuation within release names is
		// normalized equally for video/subtitle, never used as a torrent identity.
		foldStem := func(s string) string {
			return strings.Join(strings.FieldsFunc(s, func(r rune) bool { return r == '.' || r == '_' || r == ' ' || r == '-' }), " ")
		}
		candidateStem := strings.Join(tokens, " ")
		if candidateStem == foldStem(stem) {
			row.Match = "exact"
		} else {
			_, vp, vs, ve, vok := parseEpisodeIdentity(videoPath, true)
			_, sp, ss, se, sok := parseEpisodeIdentity(path.Join(dir, strings.TrimSuffix(path.Base(value), ext)+".mkv"), true)
			if !vok || !sok || vp != sp || vs != ss || ve != se {
				continue
			}
			row.Match = "episode"
			count := 0
			for _, other := range files {
				od, op, os, oe, ok := parseEpisodeIdentity(normalizedSubtitlePath(other.Path), true)
				if ok && od == dir && op == vp && os == vs && oe == ve {
					count++
				}
			}
			if count != 1 {
				row.Match = "ambiguous"
			}
		}
		if len(out) == 100 {
			return out, true
		}
		out = append(out, row)
	}
	return out, truncated
}
func decodeSubtitle(input []byte) (string, error) {
	if len(input) > SubtitleInputLimit {
		return "", ErrSubtitle
	}
	var decoded []byte
	var err error
	switch {
	case bytes.HasPrefix(input, []byte{0xff, 0xfe}):
		if len(input)%2 != 0 {
			return "", ErrSubtitle
		}
		decoded, err = textunicode.UTF16(textunicode.LittleEndian, textunicode.ExpectBOM).NewDecoder().Bytes(input)
	case bytes.HasPrefix(input, []byte{0xfe, 0xff}):
		if len(input)%2 != 0 {
			return "", ErrSubtitle
		}
		decoded, err = textunicode.UTF16(textunicode.BigEndian, textunicode.ExpectBOM).NewDecoder().Bytes(input)
	case utf8.Valid(input):
		decoded = bytes.TrimPrefix(input, []byte{0xef, 0xbb, 0xbf})
	default:
		decoded, err = charmap.Windows1251.NewDecoder().Bytes(input)
	}
	if err != nil || !utf8.Valid(decoded) {
		return "", ErrSubtitle
	}
	for _, r := range string(decoded) {
		if r == 0 || r < 32 && r != '\r' && r != '\n' && r != '\t' {
			return "", ErrSubtitle
		}
	}
	return strings.ReplaceAll(strings.ReplaceAll(string(decoded), "\r\n", "\n"), "\r", "\n"), nil
}
func subtitleTime(value string) (int64, error) {
	parts := strings.Split(strings.ReplaceAll(strings.TrimSpace(value), ",", "."), ":")
	if len(parts) == 2 {
		parts = append([]string{"0"}, parts...)
	}
	if len(parts) != 3 {
		return 0, ErrSubtitle
	}
	h, e1 := strconv.Atoi(parts[0])
	m, e2 := strconv.Atoi(parts[1])
	seconds := strings.Split(parts[2], ".")
	if len(seconds) != 2 || len(seconds[1]) < 1 || len(seconds[1]) > 3 {
		return 0, ErrSubtitle
	}
	s, e3 := strconv.Atoi(seconds[0])
	fraction, e4 := strconv.Atoi(seconds[1] + strings.Repeat("0", 3-len(seconds[1])))
	if e1 != nil || e2 != nil || e3 != nil || e4 != nil || h < 0 || h > 999 || m < 0 || m > 59 || s < 0 || s > 59 || fraction < 0 {
		return 0, ErrSubtitle
	}
	return ((int64(h)*60+int64(m))*60+int64(s))*1000 + int64(fraction), nil
}
func vttTime(ms int64) string {
	return fmt.Sprintf("%02d:%02d:%02d.%03d", ms/3600000, ms/60000%60, ms/1000%60, ms%1000)
}

var subtitleBlocks = regexp.MustCompile(`\n[ \t]*\n`)
var assOverrides = regexp.MustCompile(`\{[^}]*\}`)

func SubtitleWebVTT(input []byte, extension string) ([]byte, error) {
	return SubtitleWebVTTContext(context.Background(), input, extension)
}
func SubtitleWebVTTContext(ctx context.Context, input []byte, extension string) ([]byte, error) {
	return convertSubtitle(ctx, input, extension, true)
}

// Original delivery validates the same supported text grammar, but does not
// allocate a converted output or discard the original ASS/SSA styling.
func ValidateSubtitleContext(ctx context.Context, input []byte, extension string) error {
	_, err := convertSubtitle(ctx, input, extension, false)
	return err
}
func convertSubtitle(ctx context.Context, input []byte, extension string, render bool) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	extension = strings.ToLower(extension)
	text, err := decodeSubtitle(input)
	if err != nil {
		return nil, err
	}
	var out strings.Builder
	out.WriteString("WEBVTT\n\n")
	count := 0
	emit := func(start, end, payload string) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		from, err := subtitleTime(start)
		if err != nil {
			return err
		}
		to, err := subtitleTime(end)
		if err != nil || to <= from {
			return ErrSubtitle
		}
		if len(payload) > 256<<10 {
			return ErrSubtitle
		}
		if !render {
			count++
			if count > 100000 {
				return ErrSubtitle
			}
			return nil
		}
		payload = strings.ReplaceAll(payload, "-->", "→")
		out.WriteString(vttTime(from) + " --> " + vttTime(to) + "\n" + html.EscapeString(payload) + "\n\n")
		count++
		if out.Len() > SubtitleOutputLimit || count > 100000 {
			return ErrSubtitle
		}
		return nil
	}
	switch strings.ToLower(extension) {
	case ".srt", ".vtt":
		if extension == ".vtt" {
			if !strings.HasPrefix(text, "WEBVTT") {
				return nil, ErrSubtitle
			}
			if at := strings.Index(text, "\n\n"); at >= 0 {
				text = text[at+2:]
			} else {
				return nil, ErrSubtitle
			}
		}
		for _, block := range subtitleBlocks.Split(strings.TrimSpace(text), -1) {
			lines := strings.Split(block, "\n")
			if extension == ".vtt" && (strings.HasPrefix(block, "NOTE") || strings.HasPrefix(block, "STYLE") || strings.HasPrefix(block, "REGION")) {
				continue
			}
			at := 0
			if len(lines) > 1 && !strings.Contains(lines[0], "-->") {
				at = 1
			}
			if at >= len(lines) {
				return nil, ErrSubtitle
			}
			timing := strings.SplitN(lines[at], "-->", 2)
			if len(timing) != 2 {
				return nil, ErrSubtitle
			}
			end := strings.Fields(timing[1])
			if len(end) == 0 {
				return nil, ErrSubtitle
			}
			if err := emit(timing[0], end[0], strings.Join(lines[at+1:], "\n")); err != nil {
				return nil, err
			}
		}
	case ".ass", ".ssa":
		scanner := bufio.NewScanner(strings.NewReader(text))
		scanner.Buffer(make([]byte, 4096), 256<<10)
		fields := []string{"layer", "start", "end", "style", "name", "marginl", "marginr", "marginv", "effect", "text"}
		events := false
		for scanner.Scan() {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			line := strings.TrimSpace(scanner.Text())
			if strings.HasPrefix(line, "[") {
				events = strings.EqualFold(line, "[Events]")
				continue
			}
			if !events {
				continue
			}
			if strings.HasPrefix(strings.ToLower(line), "format:") {
				fields = strings.Split(strings.ToLower(strings.TrimSpace(line[7:])), ",")
				for i := range fields {
					fields[i] = strings.TrimSpace(fields[i])
				}
				if len(fields) > 32 || fields[len(fields)-1] != "text" {
					return nil, ErrSubtitle
				}
			}
			if strings.HasPrefix(strings.ToLower(line), "dialogue:") {
				values := strings.SplitN(strings.TrimSpace(line[9:]), ",", len(fields))
				if len(values) != len(fields) {
					return nil, ErrSubtitle
				}
				get := func(name string) string {
					for i, f := range fields {
						if f == name {
							return values[i]
						}
					}
					return ""
				}
				payload := assOverrides.ReplaceAllString(get("text"), "")
				payload = strings.NewReplacer(`\N`, "\n", `\n`, "\n", `\h`, " ").Replace(payload)
				if err := emit(get("start"), get("end"), payload); err != nil {
					return nil, err
				}
			}
		}
		if scanner.Err() != nil {
			return nil, ErrSubtitle
		}
	default:
		return nil, ErrSubtitle
	}
	if count == 0 {
		return nil, ErrSubtitle
	}
	return []byte(out.String()), nil
}
