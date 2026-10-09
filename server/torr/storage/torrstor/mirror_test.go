package torrstor

import (
	"bytes"
	"server/settings"
	"testing"
)

func TestMirrorBytesWaitForHashAndFailureFencePreservesBackend(t *testing.T) {
	old := settings.BTsets()
	settings.StoreBTsets(&settings.BTSets{CacheSize: 1 << 20})
	t.Cleanup(func() { settings.StoreBTsets(old) })
	s := NewStorage()
	hash := mkHash(0x91)
	s.RequireVerifiedReads(hash)
	s.callbackOpen(51, hash, 2, 2*pieceBlockSize)
	s.callbackSize(51, 4*pieceBlockSize)
	c := s.CacheByHash(hash)
	data := bytes.Repeat([]byte{0x31}, 2*pieceBlockSize)
	if _, err := s.callbackWrite(51, 0, 0, data); err != nil {
		t.Fatal(err)
	}
	if c.readableAt(0, 0) != 0 {
		t.Fatal("unverified mirror bytes exposed")
	}
	// Native hash failure locks the picker until this callback returns. It clears
	// readable state without removing data still owned by disk/hash callbacks.
	s.callbackClearPiece(51, 0)
	output := make([]byte, len(data))
	if _, err := s.callbackRead(51, 0, 0, output); err != nil || !bytes.Equal(output, data) {
		t.Fatal("failure fence destroyed backend")
	}
	if c.Have(0) || c.readableAt(0, 0) != 0 {
		t.Fatal("failed piece stayed readable")
	}
	if _, err := s.callbackWrite(51, 0, 0, data[:pieceBlockSize]); err != nil {
		t.Fatal(err)
	}
	if c.readableAt(0, 0) != 0 {
		t.Fatal("retry bytes exposed before a successful hash")
	}
	// A successful native hash also verifies unchanged blocks in a partial retry.
	c.SignalPieceComplete(0)
	if !c.Have(0) || c.readableAt(0, 0) != int64(len(data)) {
		t.Fatal("successful native verification failed to restore reads")
	}
	if n, err := c.readStreamPiece(0, 0, output); err != nil || n != len(data) || !bytes.Equal(output, data) {
		t.Fatal("verified stream copy failed", n, err)
	}
	// A reader may have inspected availability before the native failure fence.
	// Its final copy must recheck atomically rather than trust that old length.
	s.callbackClearPiece(51, 0)
	if n, _ := c.readStreamPiece(0, 0, output); n != 0 {
		t.Fatal("stale availability exposed the failed backend")
	}
	if n, err := s.callbackRead(51, 0, 0, output); err != nil || n != len(data) {
		t.Fatal("hash callback lost access to its retained raw backend", n, err)
	}
}

func TestOldStorageCloseCannotRemoveReplacementCacheOrMirrorGate(t *testing.T) {
	s := NewStorage()
	hash := mkHash(0x92)
	s.RequireVerifiedReads(hash)
	s.callbackOpen(61, hash, 1, pieceBlockSize)
	s.callbackOpen(62, hash, 1, pieceBlockSize)
	current := s.CacheByHash(hash)
	s.callbackClose(61)
	if s.CacheByHash(hash) != current || s.lookup(62) != current {
		t.Fatal("old native disk close removed the replacement cache")
	}
	if !current.verifiedReads.Load() || !s.verifiedReads[hash] {
		t.Fatal("old native disk close removed the replacement mirror gate")
	}
	s.callbackClose(62)
	if s.CacheByHash(hash) != nil || s.verifiedReads[hash] {
		t.Fatal("current native disk close failed to release its registry state")
	}
}
