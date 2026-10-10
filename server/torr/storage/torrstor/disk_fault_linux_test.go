package torrstor

import (
	"bytes"
	"errors"
	"syscall"
	"testing"
)

func TestFullDiskPropagatesENOSPCWithoutAdvertisingBlocks(t *testing.T) {
	withDiskCache(t, 0)
	s := NewStorage()
	s.callbackOpen(1, mkHash(0x73), 2, pieceLen)
	s.callbackSize(1, 2*pieceLen)
	t.Cleanup(func() { s.callbackClose(1) })
	p := newPiece(s.lookup(1), 0)
	// Linux's real full-device exercises WriteAt's ENOSPC result. Skip initial
	// regular-file truncation so the test reaches the write syscall itself.
	p.disk.name, p.disk.initialized = "/dev/full", true
	n, err := p.WriteAt(bytes.Repeat([]byte{0x73}, pieceBlockSize), 0)
	if n != 0 || !errors.Is(err, syscall.ENOSPC) {
		t.Fatalf("full disk: n=%d error=%v", n, err)
	}
	if p.availableFrom(0) != 0 || p.size != 0 || p.markVerifiedComplete() {
		t.Fatal("ENOSPC advertised unwritten bytes")
	}
	p.disk.Close()
}
