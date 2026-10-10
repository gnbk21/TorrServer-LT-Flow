package lt

import (
	"bytes"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
)

func protocolFixture(hybrid bool) ([]byte, string, string) {
	data := bytes.Repeat([]byte{0x71}, 10001)
	root := sha256.Sum256(data)
	info := fmt.Sprintf("d9:file treed11:episode.mkvd0:d6:lengthi10001e11:pieces root32:%seee", string(root[:]))
	if hybrid {
		info += "6:lengthi10001e"
	}
	info += "12:meta versioni2e4:name11:episode.mkv12:piece lengthi32768e"
	if hybrid {
		piece := sha1.Sum(data)
		info += "6:pieces20:" + string(piece[:])
	}
	info += "e"
	v1 := sha1.Sum([]byte(info))
	v2 := sha256.Sum256([]byte(info))
	return []byte("d4:info" + info + "e"), hex.EncodeToString(v1[:]), hex.EncodeToString(v2[:])
}

func TestNativeFullV2AndHybridIdentities(t *testing.T) {
	for _, hybrid := range []bool{false, true} {
		t.Run(fmt.Sprint(hybrid), func(t *testing.T) {
			payload, v1, v2 := protocolFixture(hybrid)
			parsed, err := ParseTorrentBytes(payload)
			if err != nil {
				t.Fatal(err)
			}
			want := v2
			if hybrid {
				want = v1
			}
			if parsed.InfoHash != want || parsed.InfoHashV2 != v2 || len(parsed.V2Files) != 1 {
				t.Fatalf("lost full identity or file roots: %+v", parsed)
			}
			if hybrid && parsed.InfoHashV1 != v1 || !hybrid && parsed.InfoHashV1 != "" {
				t.Fatal("changed v1 identity")
			}
			s := newSession(t)
			h, err := s.AddTorrent(AddTorrentParams{InfoBytes: payload, SavePath: t.TempDir(), Paused: true})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = h.Remove(false) })
			st, err := h.Status()
			if err != nil {
				t.Fatal(err)
			}
			if h.InfoHash() != want || st.InfoHash != want || st.InfoHashV2 != v2 {
				t.Fatal("native accessor truncated a digest", st)
			}
			viaMagnet, err := s.AddTorrent(AddTorrentParams{Link: "magnet:?xt=urn:btmh:1220" + v2, SavePath: t.TempDir(), Paused: true})
			if err != nil || viaMagnet.ID() != h.ID() {
				t.Fatal("hybrid/v2 alias duplicated a native handle", err)
			}
			if files, err := h.Files(); err != nil || len(files) != 1 || files[0].Index != 0 {
				t.Fatal("padding became a user file", files, err)
			}
		})
	}
}

func TestNativeV2MagnetsKeepDistinctFullDigests(t *testing.T) {
	s := newSession(t)
	var ids []int64
	for _, hash := range []string{strings.Repeat("ab", 32), strings.Repeat("ab", 20) + strings.Repeat("cd", 12)} {
		h, err := s.AddTorrent(AddTorrentParams{Link: "magnet:?xt=urn:btmh:1220" + hash, SavePath: t.TempDir(), Paused: true})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = h.Remove(false) })
		if h.InfoHash() != hash {
			t.Fatal("truncated v2 info hash")
		}
		ids = append(ids, h.ID())
	}
	if ids[0] == ids[1] {
		t.Fatal("distinct v2 hashes with identical SHA-1-sized prefix were aliased")
	}
}
