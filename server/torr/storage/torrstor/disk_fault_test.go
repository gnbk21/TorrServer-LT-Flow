package torrstor

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDiskWriteFailureDoesNotAdvertiseDataAndRecovers(t *testing.T) {
	dir := withDiskCache(t, 0)
	s := NewStorage()
	h := mkHash(0x71)
	const length = 4 * pieceBlockSize
	s.callbackOpen(1, h, 2, length)
	s.callbackSize(1, 2*length)
	t.Cleanup(func() { s.callbackClose(1) })
	name := filepath.Join(dir, hashHex(h), "0")
	// An actual filesystem failure, without modifying user files or permissions.
	if err := os.MkdirAll(name, 0700); err != nil {
		t.Fatal(err)
	}
	payload := bytes.Repeat([]byte{0x71}, pieceBlockSize)
	if n, err := s.callbackWrite(1, 0, 0, payload); n != 0 || err == nil {
		t.Fatalf("failed backing file write: n=%d error=%v", n, err)
	}
	c := s.lookup(1)
	events, _ := c.DrainStorageEvidence()
	if len(events) != 1 || events[0].Stage != "CACHE_WRITE_FAILED" || events[0].Bytes != 0 {
		t.Fatalf("backing file failure missing from bounded evidence: %+v", events)
	}
	c.SignalPieceComplete(0)
	if c.Have(0) || c.readableAt(0, 0) != 0 {
		t.Fatal("failed write exposed data")
	}
	if err := os.Remove(name); err != nil {
		t.Fatal(err)
	}
	if n, err := s.callbackWrite(1, 0, 0, payload); n != len(payload) || err != nil {
		t.Fatalf("write after recovery: n=%d error=%v", n, err)
	}
	if c.Have(0) || c.readableAt(0, 0) != pieceBlockSize {
		t.Fatal("recovery incorrectly completed the partial piece")
	}
	got := make([]byte, len(payload))
	if n, err := s.callbackRead(1, 0, 0, got); n != len(got) || err != nil || !bytes.Equal(got, payload) {
		t.Fatalf("recovered original bytes: n=%d error=%v", n, err)
	}
}

// Hold the real backing-file lock to model delayed storage without introducing
// a production IO hook. Other pieces must still progress, and the completed
// operation must produce measured, bounded evidence rather than a guessed cause.
func TestSlowBackingFileKeepsOtherPiecesResponsiveAndRecordsDuration(t *testing.T) {
	withDiskCache(t, 0)
	s := NewStorage()
	s.callbackOpen(1, mkHash(0x74), 2, 2*pieceBlockSize)
	s.callbackSize(1, 4*pieceBlockSize)
	t.Cleanup(func() { s.callbackClose(1) })
	payload := bytes.Repeat([]byte{0x74}, pieceBlockSize)
	if _, err := s.callbackWrite(1, 0, 0, payload); err != nil {
		t.Fatal(err)
	}
	c := s.lookup(1)
	c.mu.RLock()
	piece := c.pieces[0]
	c.mu.RUnlock()
	c.DrainStorageEvidence()
	piece.disk.mu.Lock()
	locked := true
	defer func() {
		if locked {
			piece.disk.mu.Unlock()
		}
	}()
	done := make(chan error, 1)
	go func() {
		_, err := s.callbackWrite(1, 0, pieceBlockSize, payload)
		done <- err
	}()
	deadline := time.Now().Add(time.Second)
	for piece.mu.TryLock() {
		piece.mu.Unlock()
		if time.Now().After(deadline) {
			t.Fatal("delayed writer never reached the backing file")
		}
		time.Sleep(time.Millisecond)
	}
	other := make(chan error, 1)
	go func() {
		_, err := s.callbackWrite(1, 1, 0, payload)
		other <- err
	}()
	select {
	case err := <-other:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("one delayed backing file blocked another piece")
	}
	time.Sleep(60 * time.Millisecond)
	piece.disk.mu.Unlock()
	locked = false
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("delayed writer did not recover")
	}
	if c.Have(0) {
		t.Fatal("writing all blocks bypassed native verification")
	}
	events, dropped := c.DrainStorageEvidence()
	observed := false
	for _, event := range events {
		if event.Stage == "CACHE_WRITE_SLOW" && event.OperationMeasured && event.ElapsedMs >= 50 && event.Bytes == pieceBlockSize {
			observed = true
		}
	}
	if !observed || dropped != 0 {
		t.Fatal("slow storage duration was not captured", events, dropped)
	}
	c.SignalPieceComplete(0)
	got := make([]byte, 2*pieceBlockSize)
	if n, err := s.callbackRead(1, 0, 0, got); n != len(got) || err != nil || !bytes.Equal(got, bytes.Repeat([]byte{0x74}, len(got))) {
		t.Fatal("recovery changed original bytes", n, err)
	}
}

func TestInterruptedDiskWriteDoesNotBecomeVerifiedAfterRestart(t *testing.T) {
	withDiskCache(t, 0)
	h := mkHash(0x72)
	s := NewStorage()
	const length = 4 * pieceBlockSize
	s.callbackOpen(1, h, 2, length)
	s.callbackSize(1, 2*length)
	// Truncate allocates the final length but only one non-final block is written.
	if _, err := s.callbackWrite(1, 0, pieceBlockSize, bytes.Repeat([]byte{0x72}, pieceBlockSize)); err != nil {
		t.Fatal(err)
	}
	s.callbackClose(1)
	bitmap := ScanHavePieces(h, 2, length, 2*length, nil)
	if len(bitmap) != 0 && bitmap[0] != 0 {
		t.Fatal("preallocated interrupted data was accepted without hashes")
	}
	reopened := NewStorage()
	reopened.SetVerifiedResume(h, bitmap, 2*length)
	reopened.callbackOpen(2, h, 2, length)
	reopened.callbackSize(2, 2*length)
	t.Cleanup(func() { reopened.callbackClose(2) })
	if reopened.callbackHave(2, 0) || reopened.lookup(2).readableAt(0, 0) != 0 {
		t.Fatal("restart advertised interrupted unverified bytes")
	}
}
