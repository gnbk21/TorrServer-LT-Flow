package torrstor

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
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
