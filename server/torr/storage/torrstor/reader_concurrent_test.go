package torrstor

import (
	"bytes"
	"io"
	"sync"
	"testing"
	"time"

	"server/lt"
)

func TestReaderWindowExpiryKeepsNewPreloadDemand(t *testing.T) {
	payload, _ := networkProtocolFixture(false)
	session, err := lt.NewSession(lt.SessionConfig{
		"listen_interfaces": "127.0.0.1:0", "enable_dht": false, "enable_lsd": false,
		"enable_upnp": false, "enable_natpmp": false,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	handle, err := session.AddTorrent(lt.AddTorrentParams{InfoBytes: payload, SavePath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer handle.Remove(false)
	cache := newCache(nil, 1, mkHash(0x11), 3, 32768)
	defer cache.close()
	reader := NewReader(cache, handle, FileInfo{Length: 65536})
	reader.winFirst.Store(0)
	reader.winLast.Store(1)

	if err := handle.PrioritizePieces([]int{4, 0, 0}); err != nil {
		t.Fatal(err)
	}
	want, err := handle.Status()
	if err != nil || want.TotalWanted == 0 {
		t.Fatalf("preload fixture needs piece 0: (%v, %v)", want, err)
	}
	if err := handle.PrioritizePieces([]int{4, 4, 0}); err != nil {
		t.Fatal(err)
	}
	shared, err := handle.Status()
	if err != nil || shared.TotalWanted <= want.TotalWanted {
		t.Fatalf("old window did not request the second piece: (%v, %v)", shared, err)
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
	cache.SetPreloadReserve([][2]int{{0, 0}})

	deadline := time.Now().Add(windowLingerDelay + 2*time.Second)
	for time.Now().Before(deadline) {
		status, err := handle.Status()
		if err != nil {
			t.Fatal(err)
		}
		if status.TotalWanted == want.TotalWanted {
			return
		}
		if status.TotalWanted < want.TotalWanted {
			t.Fatalf("expired reader demoted another owner's piece: wanted %d bytes, want %d", status.TotalWanted, want.TotalWanted)
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("expired reader's unclaimed piece kept downloading")
}

// TestReader_ConcurrentReadersOnSameCache exercises the FUSE / WebDAV
// access pattern: several handles (== several Readers) read the same
// file in parallel. They share the Cache and the underlying piece
// storage; the test asserts they don't trip over each other.
func TestReader_ConcurrentReadersOnSameCache(t *testing.T) {
	const filePieces = 4
	c := mkCache(t, int64(filePieces)*pieceLen)

	// Fill all pieces deterministically: piece i is byte-value 'A'+i, then
	// signal completion (production gets this from piece_finished_alert).
	for i := 0; i < filePieces; i++ {
		payload := bytes.Repeat([]byte{byte('A' + i)}, int(pieceLen))
		if _, err := c.writePiece(i, 0, payload); err != nil {
			t.Fatalf("writePiece(%d): %v", i, err)
		}
		c.SignalPieceComplete(i)
	}

	expect := make([]byte, int(int64(filePieces)*pieceLen))
	for i := 0; i < filePieces; i++ {
		copy(expect[i*int(pieceLen):(i+1)*int(pieceLen)],
			bytes.Repeat([]byte{byte('A' + i)}, int(pieceLen)))
	}

	const workers = 8
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			r := NewReader(c, nil, FileInfo{Offset: 0, Length: int64(filePieces) * pieceLen})
			defer r.Close()
			got, err := io.ReadAll(r)
			if err != nil {
				errs <- err
				return
			}
			if !bytes.Equal(got, expect) {
				errs <- io.ErrUnexpectedEOF
			}
		}(w)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent worker: %v", err)
		}
	}

	if got := c.ActiveReaders(); got != 0 {
		t.Fatalf("readers left registered after Close: %d", got)
	}
}

// TestReader_SatisfiesReadSeekCloser pins the type assertion that
// FUSE (server/torrfs/fuse) and WebDAV (server/torrfs/webdav) make on
// the value returned by our wrapper. If anybody removes Seek/Close from
// torrstor.Reader, this fails at compile time.
func TestReader_SatisfiesReadSeekCloser(t *testing.T) {
	var _ io.ReadSeekCloser = (*Reader)(nil)
	_ = t
}

// TestReader_StatBeforeReadWorks mirrors what os.File / fs.Stat callers
// do (and what WebDAV's ReadOnlyFS.Stat does before OpenFile): obtain
// size from the Reader without consuming data.
func TestReader_StatBeforeReadWorks(t *testing.T) {
	c := mkCache(t, pieceLen)
	_, _ = c.writePiece(0, 0, bytes.Repeat([]byte{0x77}, int(pieceLen)))
	c.SignalPieceComplete(0)
	r := NewReader(c, nil, FileInfo{Offset: 0, Length: pieceLen})
	defer r.Close()

	// Reader doesn't implement Stat itself; the caller uses fs.Stat()
	// against the surrounding FS. Here we just verify that Seek(0, End)
	// reports the file length without disturbing the read position.
	end, err := r.Seek(0, io.SeekEnd)
	if err != nil {
		t.Fatalf("Seek end: %v", err)
	}
	if end != pieceLen {
		t.Fatalf("SeekEnd: got %d, want %d", end, pieceLen)
	}
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		t.Fatalf("Seek start: %v", err)
	}
	buf := make([]byte, pieceLen)
	if _, err := io.ReadFull(r, buf); err != nil {
		t.Fatalf("read after rewind: %v", err)
	}
}
