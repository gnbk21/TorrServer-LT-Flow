package lt

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestPrivateMetadataProtectsAuthorizedTrackerTiers(t *testing.T) {
	var authorized, extra atomic.Int32
	tracker := func(counter *atomic.Int32) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			counter.Add(1)
			_, _ = w.Write([]byte("d8:intervali1800e5:peers0:e"))
		}))
	}
	a, b := tracker(&authorized), tracker(&extra)
	defer a.Close()
	defer b.Close()
	aURL, bURL := a.URL+"/announce", b.URL+"/announce"
	encode := func(url string) string { return fmt.Sprintf("%d:%s", len(url), url) }
	info := "d6:lengthi100e4:name4:test12:piece lengthi16384e6:pieces20:" + strings.Repeat("\x00", 20) + "7:privatei1ee"
	data := []byte("d8:announce" + encode(aURL) + "13:announce-listll" + encode(aURL) + "el" + encode(bURL) + "ee4:info" + info + "e")
	parsed, err := ParseTorrentBytes(data)
	if err != nil {
		t.Fatal(err)
	}
	if !parsed.Private || len(parsed.TrackerTiers) != 2 || parsed.TrackerTiers[0][0] != aURL || parsed.TrackerTiers[1][0] != bURL {
		t.Fatalf("private tiers lost: %+v", parsed)
	}
	s := newSession(t)
	if err := s.ApplySettings(SessionConfig{"announce_to_all_tiers": true, "announce_to_all_trackers": true}); err != nil {
		t.Fatal(err)
	}
	torrent, err := s.AddTorrent(AddTorrentParams{InfoBytes: data, TrackerTiers: [][]string{{bURL}}, SavePath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer torrent.Remove(false)
	if err := torrent.ReplaceTrackers([][]string{{bURL}}); err == nil {
		t.Fatal("private tracker replacement allowed")
	}
	deadline := time.Now().Add(5 * time.Second)
	for authorized.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if authorized.Load() == 0 {
		t.Fatal("authorized tracker never announced")
	}
	time.Sleep(300 * time.Millisecond)
	if extra.Load() != 0 {
		t.Fatal("private torrent announced a second tracker despite healthy first tier")
	}
	if status, err := torrent.Status(); err != nil || !status.Private {
		t.Fatalf("private status missing: %+v %v", status, err)
	}
}

func TestSparseSnapshotIsBoundedAndPrivate(t *testing.T) {
	s := newSession(t)
	torrent, err := s.AddTorrent(AddTorrentParams{InfoBytes: minimalTorrent(), SavePath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer torrent.Remove(false)
	if _, err := torrent.SampleSparse([][2]int{{0, 129}}); err == nil {
		t.Fatal("oversized snapshot accepted")
	}
	for i := 0; i < 30; i++ {
		snapshot, err := torrent.SampleSparse([][2]int{{0, 1}})
		if err != nil {
			t.Fatal(err)
		}
		if snapshot.Known {
			if len(snapshot.Windows) != 1 || len(snapshot.Windows[0].Availability) != 1 || snapshot.SampledPeers != 0 {
				t.Fatalf("bad bounded snapshot: %+v", snapshot)
			}
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("asynchronous snapshot was never published")
}
