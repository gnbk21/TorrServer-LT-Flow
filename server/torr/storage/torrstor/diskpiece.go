package torrstor

import (
	"io"
	"os"
	"path/filepath"
	"strconv"
	"sync"
)

// DiskPiece persists a single piece to its own file on disk at
// `<savePath>/<infoHashHex>/<pieceID>`. Preserves the legacy
// piece-per-file layout so cache directories from the anacrolix era
// load unchanged.
//
// The file is lazily created on first write. ReadAt against a missing
// file returns io.EOF.
type DiskPiece struct {
	piece       *Piece
	dir         string
	name        string
	initialized bool

	mu sync.RWMutex
}

func newDiskPiece(p *Piece, savePath string) *DiskPiece {
	dir := filepath.Join(savePath, hashHex(p.cache.InfoHash))
	name := filepath.Join(dir, strconv.Itoa(p.Id))
	dp := &DiskPiece{piece: p, dir: dir, name: name}
	// Detect existing file from a previous run so the scan-resume path
	// reports a sane initial size.
	if fi, err := os.Stat(name); err == nil && p.Id/8 < len(p.cache.resume) &&
		p.cache.resume[p.Id/8]&(1<<uint(p.Id%8)) != 0 && fi.Size() == p.expectedSize() {
		p.size = fi.Size()
		p.complete = true
		p.accessed.Store(fi.ModTime().Unix())
	}
	return dp
}

func (dp *DiskPiece) WriteAt(b []byte, off int64) (int, error) {
	if len(b) == 0 {
		return 0, nil
	}
	dp.mu.Lock()
	defer dp.mu.Unlock()
	f, done, err := dp.piece.cache.diskFiles.acquire(dp.name, true)
	if err != nil {
		return 0, err
	}
	defer done()
	if !dp.initialized {
		if err := f.Truncate(dp.piece.expectedSize()); err != nil {
			return 0, err
		}
		dp.initialized = true
	}
	return f.WriteAt(b, off)
}

func (dp *DiskPiece) ReadAt(b []byte, off int64) (int, error) {
	if len(b) == 0 {
		return 0, nil
	}
	dp.mu.RLock()
	defer dp.mu.RUnlock()
	f, done, err := dp.piece.cache.diskFiles.acquire(dp.name, false)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, io.EOF
		}
		return 0, err
	}
	defer done()
	n, err := f.ReadAt(b, off)
	if err == io.EOF && n > 0 {
		err = nil
	}
	return n, err
}

// Release removes the on-disk file.
func (dp *DiskPiece) Release() {
	dp.mu.Lock()
	defer dp.mu.Unlock()
	dp.piece.cache.diskFiles.closePath(dp.name)
	_ = os.Remove(dp.name)
	dp.initialized = false
}

func (dp *DiskPiece) Close() {
	dp.mu.Lock()
	defer dp.mu.Unlock()
	dp.piece.cache.diskFiles.closePath(dp.name)
}

func (dp *DiskPiece) Sync() error {
	dp.mu.Lock()
	defer dp.mu.Unlock()
	// Windows FlushFileBuffers requires write access, including resumed pieces
	// whose pooled handle was first opened by a read. Never create missing data.
	f, done, err := dp.piece.cache.diskFiles.acquireMode(dp.name, true, false)
	if err != nil {
		return err
	}
	defer done()
	return f.Sync()
}
