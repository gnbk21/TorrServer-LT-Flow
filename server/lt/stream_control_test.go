package lt

import (
	"errors"
	"testing"
	"time"
)

func TestStreamFrontierValidationExpiryAndNativeObservations(t *testing.T) {
	s := newSession(t)
	tor, err := s.AddTorrent(AddTorrentParams{InfoBytes: minimalTorrent(), SavePath: t.TempDir(), Paused: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, pieces := range [][]int{{-1}, {1}, make([]int, 9)} {
		if err := tor.SetStreamFrontiers(pieces); !errors.Is(err, ErrInvalid) {
			t.Fatalf("invalid frontier %v: %v", pieces, err)
		}
	}
	if err := tor.SetStreamFrontiers([]int{0, 0}); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	seen := false
	for time.Now().Before(deadline) {
		snapshot, err := tor.SampleSparse(nil)
		if err != nil {
			t.Fatal(err)
		}
		if snapshot.Stream != nil && snapshot.Stream.Known && snapshot.Stream.Frontiers == 1 {
			seen = true
			if snapshot.Stream.FirstConnectMs != -1 || snapshot.Stream.ConnectDurationMs != -1 ||
				len(snapshot.Stream.Service) != 0 || !snapshot.Stream.Valid() {
				t.Fatalf("invented connection evidence: %+v", snapshot.Stream)
			}
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !seen {
		t.Fatal("frontier never reached native engine")
	}
	// A paused torrent has no scheduler ticks. Expiry must remain observable
	// even without a second Go update or an active reader.
	deadline = time.Now().Add(6 * time.Second)
	for time.Now().Before(deadline) {
		snapshot, err := tor.SampleSparse(nil)
		if err != nil {
			t.Fatal(err)
		}
		if snapshot.Stream != nil && snapshot.Stream.Known && snapshot.Stream.Frontiers == 0 {
			if err := tor.SetStreamFrontiers(nil); err != nil {
				t.Fatal(err)
			}
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("stale frontier did not expire")
}
