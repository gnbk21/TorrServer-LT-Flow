package lt

import (
	"bytes"
	"strings"
	"testing"
)

func TestNativeMetadataFallbackValidatesIdentityAndKeepsHandle(t *testing.T) {
	for _, hybrid := range []bool{false, true} {
		payload, v1, v2 := protocolFixture(hybrid)
		magnet := "magnet:?xt=urn:btmh:1220" + v2
		if hybrid {
			magnet += "&xt=urn:btih:" + v1
		}
		s := newSession(t)
		h, err := s.AddTorrent(AddTorrentParams{Link: magnet, SavePath: t.TempDir(), Paused: true,
			TrackerTiers: [][]string{{"https://authorized.example/announce"}}})
		if err != nil {
			t.Fatal(err)
		}
		id := h.ID()
		for _, invalid := range [][]byte{nil, []byte("<html>blocked</html>"), bytes.ReplaceAll(payload, []byte("episode.mkv"), []byte("another.mkv"))} {
			if installed, err := h.SetMetadata(invalid); err == nil || installed {
				t.Fatal("accepted absent, invalid or mismatched metadata")
			}
		}
		if have, _ := h.HaveMetadata(); have {
			t.Fatal("rejected source changed metadata")
		}
		installed, err := h.SetMetadata(payload)
		if err != nil || !installed || h.ID() != id {
			t.Fatal("valid metadata did not reuse native handle", installed, err)
		}
		if have, err := h.HaveMetadata(); err != nil || !have {
			t.Fatal("installed metadata unavailable", err)
		}
		if installed, err := h.SetMetadata(payload); err != nil || installed {
			t.Fatal("already available metadata was reported as newly recovered", err)
		}
		if trackers := strings.Join(h.Trackers(), ","); trackers != "https://authorized.example/announce" {
			t.Fatal("changed authorized tracker tiers", trackers)
		}
		if _, err := h.Metadata(); err != nil {
			t.Fatal("metadata cannot be persisted", err)
		}
		_ = h.Remove(false)
		if _, err := h.SetMetadata(payload); err == nil {
			t.Fatal("used retired handle")
		}
	}
}

func TestNativeMetadataFallbackRejectsWrongSecondHybridHash(t *testing.T) {
	payload, v1, v2 := protocolFixture(true)
	s := newSession(t)
	wrongV2 := strings.Repeat("ab", 32)
	if wrongV2 == v2 {
		t.Fatal("fixture collision")
	}
	h, err := s.AddTorrent(AddTorrentParams{Link: "magnet:?xt=urn:btih:" + v1 + "&xt=urn:btmh:1220" + wrongV2, SavePath: t.TempDir(), Paused: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.SetMetadata(payload); err == nil {
		t.Fatal("checked only the first hybrid hash")
	}
}

func TestNativeMetadataFallbackIgnoresOuterSourcesForPrivateTorrent(t *testing.T) {
	info := "d6:lengthi100e4:name4:test12:piece lengthi16384e6:pieces20:" + strings.Repeat("\x00", 20) + "7:privatei1ee"
	payload := []byte("d8:announce32:https://untrusted.test/announce/4:info" + info + "e")
	parsed, err := ParseTorrentBytes(payload)
	if err != nil || !parsed.Private {
		t.Fatal("invalid private fixture", err)
	}
	s := newSession(t)
	h, err := s.AddTorrent(AddTorrentParams{Link: "magnet:?xt=urn:btih:" + parsed.InfoHash,
		SavePath: t.TempDir(), Paused: true, TrackerTiers: [][]string{{"https://authorized.example/announce"}}})
	if err != nil {
		t.Fatal(err)
	}
	if installed, err := h.SetMetadata(payload); err != nil || !installed {
		t.Fatal("private metadata rejected", err)
	}
	if st, err := h.Status(); err != nil || !st.Private {
		t.Fatal("private flag lost", err)
	}
	if trackers := strings.Join(h.Trackers(), ","); trackers != "https://authorized.example/announce" {
		t.Fatal("outer tracker replaced the owner's authorized tracker", trackers)
	}
}
