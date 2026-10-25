package torrstor

import (
	"bytes"
	"errors"
	"os"
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
	// A real full-device descriptor exercises WriteAt's ENOSPC result. Inject
	// that descriptor into the pool: production path validation correctly rejects
	// device files and escaping symlinks, so changing the pathname cannot bypass it.
	file, err := os.OpenFile("/dev/full", os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	pool := p.cache.diskFiles
	pool.mu.Lock()
	pool.files[p.disk.name] = &diskHandle{file: file, writable: true, element: pool.lru.PushBack(p.disk.name)}
	pool.mu.Unlock()
	p.disk.initialized = true
	n, err := p.WriteAt(bytes.Repeat([]byte{0x73}, pieceBlockSize), 0)
	if n != 0 || !errors.Is(err, syscall.ENOSPC) {
		t.Fatalf("full disk: n=%d error=%v", n, err)
	}
	if p.availableFrom(0) != 0 || p.size != 0 || p.markVerifiedComplete() {
		t.Fatal("ENOSPC advertised unwritten bytes")
	}
	p.disk.Close()
}
