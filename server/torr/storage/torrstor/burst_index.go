package torrstor

import (
	"io"
	"path/filepath"
	"strings"
	"time"

	"server/flow"
	"server/settings"
)

type cachedBurstIndex struct {
	file      FileInfo
	index     flow.BurstIndex
	attempted time.Time
}
type residentIndexReader struct {
	cache *Cache
	file  FileInfo
}

func (r residentIndexReader) ReadAt(dst []byte, off int64) (int, error) {
	if off < 0 || off > r.file.Length || int64(len(dst)) > r.file.Length-off {
		return 0, io.EOF
	}
	n := 0
	for n < len(dst) {
		abs := r.file.Offset + off + int64(n)
		piece, at := int(abs/r.cache.PieceLength), abs%r.cache.PieceLength
		if !r.cache.Have(piece) {
			return n, io.EOF
		}
		count := min(len(dst)-n, int(r.cache.PieceLength-at))
		got, err := r.cache.readStreamPiece(piece, at, dst[n:n+count])
		n += got
		if err != nil || got != count {
			return n, io.EOF
		}
	}
	return n, nil
}

func (c *Cache) requestBurstIndex(file FileInfo) {
	f := settings.CurrentFlow()
	if !f.Enabled || !f.ContainerBurstHints || file.Length < 16 || c.PieceLength <= 0 {
		return
	}
	switch strings.ToLower(filepath.Ext(file.Path)) {
	case ".mkv", ".webm", ".mp4", ".m4v", ".mov":
	default:
		return
	}
	c.indexMu.Lock()
	defer c.indexMu.Unlock()
	if c.indexClosed || c.indexBusy {
		return
	}
	if c.indexes == nil {
		c.indexes = make(map[int]cachedBurstIndex)
	}
	entry, exists := c.indexes[file.Index]
	if exists && (len(entry.index.Points) > 0 || time.Since(entry.attempted) < time.Minute) {
		return
	}
	if !exists && len(c.indexes) >= 16 {
		oldest := file.Index
		var at time.Time
		for id, e := range c.indexes {
			if at.IsZero() || e.attempted.Before(at) {
				oldest, at = id, e.attempted
			}
		}
		delete(c.indexes, oldest)
	}
	c.indexes[file.Index] = cachedBurstIndex{file: file, attempted: time.Now()}
	c.indexBusy = true
	c.indexWorkers.Add(1)
	go func() {
		defer c.indexWorkers.Done()
		index, _ := flow.ReadBurstIndex(residentIndexReader{c, file}, file.Length)
		c.indexMu.Lock()
		defer c.indexMu.Unlock()
		if !c.indexClosed {
			entry := c.indexes[file.Index]
			entry.index = index
			c.indexes[file.Index] = entry
		}
		c.indexBusy = false
	}()
}

func (c *Cache) burstDemand(file int, anchor int, average float64) (float64, string) {
	c.indexMu.Lock()
	defer c.indexMu.Unlock()
	entry, ok := c.indexes[file]
	if !ok || len(entry.index.Points) == 0 {
		return average, "unknown"
	}
	return entry.index.Rate(int64(anchor)*c.PieceLength-entry.file.Offset, average), entry.index.Source
}

func (c *Cache) stopBurstIndexes() {
	c.indexMu.Lock()
	c.indexClosed = true
	c.indexMu.Unlock()
	c.indexWorkers.Wait()
	c.indexMu.Lock()
	c.indexes = nil
	c.indexMu.Unlock()
}
