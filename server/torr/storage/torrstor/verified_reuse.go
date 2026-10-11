package torrstor

import (
	"context"
	"server/flow"
	"server/lt"
	"server/settings"
	"sync/atomic"
	"time"
)

const reuseDescriptorLimit = 512
const reuseLayerBudget = 2 << 20

type reuseEntry struct {
	cache *Cache
	file  *flow.ReuseFile
}

var reuseWorker atomic.Bool

// SetReuseMetadata replaces an owner's metadata atomically. The index retains
// only authenticated roots/layers, bounded across all live caches. Private,
// incomplete or oversized metadata yields ordinary swarm downloads.
func (s *Storage) SetReuseMetadata(c *Cache, metadata *lt.ParsedTorrent) bool {
	if c == nil || c.storage != s {
		return false
	}
	s.reuseMu.Lock()
	defer s.reuseMu.Unlock()
	s.removeReuseLocked(c)
	if metadata == nil || c.reuseClosed.Load() || metadata.Private || flow.ParseInfoHash(metadata.InfoHash) != c.InfoHash || metadata.InfoHashV2 == "" || !metadata.CompletePieceLayers() || metadata.PieceLength != c.PieceLength || len(metadata.V2Files) > reuseDescriptorLimit {
		return false
	}
	entries := make([]reuseEntry, 0, len(metadata.V2Files))
	layerBytes := 0
	lastEnd := int64(0)
	for _, file := range metadata.V2Files {
		if file.Offset < lastEnd {
			return false
		}
		proof := flow.NewReuseFile(file, metadata.PieceLength, metadata.TotalSize)
		if proof == nil {
			return false
		}
		lastEnd = file.Offset + file.Size
		layerBytes += len(proof.Pieces) * 32
		if len(s.reuseEntries)+len(entries)+1 > reuseDescriptorLimit || s.reuseLayerBytes+layerBytes > reuseLayerBudget {
			return false
		}
		entries = append(entries, reuseEntry{c, proof})
	}
	if len(entries) == 0 {
		return false
	}
	s.reuseEntries = append(s.reuseEntries, entries...)
	s.reuseLayerBytes += layerBytes
	return true
}

func (s *Storage) removeReuseLocked(c *Cache) {
	kept := s.reuseEntries[:0]
	for _, entry := range s.reuseEntries {
		if entry.cache == c {
			s.reuseLayerBytes -= len(entry.file.Pieces) * 32
		} else {
			kept = append(kept, entry)
		}
	}
	clear(s.reuseEntries[len(kept):])
	s.reuseEntries = kept
}
func (c *Cache) clearReuse() {
	c.reuseClosed.Store(true)
	if c.storage != nil {
		c.storage.reuseMu.Lock()
		c.storage.removeReuseLocked(c)
		c.storage.reuseMu.Unlock()
	}
}

func (s *Storage) reuseCandidate(c *Cache, piece int) (*Cache, *flow.ReuseFile, *flow.ReuseFile, int) {
	s.reuseMu.Lock()
	defer s.reuseMu.Unlock()
	var target *flow.ReuseFile
	index := 0
	for _, entry := range s.reuseEntries {
		if entry.cache != c {
			continue
		}
		local := piece - int(entry.file.Offset/entry.file.PieceLength)
		if entry.file.PieceBytes(local) > 0 {
			target = entry.file
			index = local
			break
		}
	}
	if target != nil {
		for _, entry := range s.reuseEntries {
			if entry.cache != c && !entry.cache.reuseClosed.Load() && target.Compatible(entry.file) && entry.cache.Have(int(entry.file.Offset/entry.file.PieceLength)+index) {
				return entry.cache, entry.file, target, index
			}
		}
	}
	return nil, nil, nil, 0
}

func (c *Cache) reuseAllowed(bytes int64) bool {
	if !settings.CurrentFlow().Enabled || c.reuseClosed.Load() || c.flowReconnect.Load() || c.backgroundLimited.Load() || bytes <= 0 || bytes > 8<<20 {
		return false
	}
	budget := globalCacheSize()
	if limit := c.resourceBudget.Load(); limit > 0 {
		budget = min(budget, limit)
	}
	if c.Filled()+bytes > budget {
		return false
	}
	if c.storage == nil {
		return false
	}
	c.flowMu.Lock()
	paused := c.flowPaused
	c.flowMu.Unlock()
	if paused {
		return false
	}
	c.storage.mu.RLock()
	defer c.storage.mu.RUnlock()
	if c.storage.byHash[c.InfoHash] != c {
		return false
	}
	for _, other := range c.storage.caches {
		if other != c && (other.flowWaiting.Load() > 0 || other.flowReconnect.Load() || time.Now().UnixMilli() < other.warmupRiskUntil.Load()) {
			return false
		}
		if other.backgroundLimited.Load() {
			return false
		}
	}
	return true
}

// TryReusePiece is also used by the existing cancellable startup preload. It
// schedules only the exact missing demand, without downloading another file.
func (c *Cache) TryReusePiece(ctx context.Context, handle *lt.Torrent, piece int) bool {
	return c.tryReusePiece(ctx, nil, handle, piece)
}

// Foreground read misses opportunistically reuse existing verified bytes. One
// worker copies <=8 MiB; the native admission bounds its own asynchronous copy.
// Ordinary peer fetching proceeds concurrently and always supplies the fallback.
func (c *Cache) tryReusePiece(ctx context.Context, closed <-chan struct{}, handle *lt.Torrent, piece int) bool {
	if handle == nil || ctx == nil || ctx.Err() != nil || c.storage == nil || !reuseWorker.CompareAndSwap(false, true) {
		return false
	}
	source, from, to, index := c.storage.reuseCandidate(c, piece)
	if source == nil || !c.reuseAllowed(to.PieceBytes(index)) {
		reuseWorker.Store(false)
		return false
	}
	go func() {
		defer reuseWorker.Store(false)
		cancelled := func() bool {
			select {
			case <-closed:
				return true
			default:
				return ctx.Err() != nil
			}
		}
		if cancelled() || source.reuseClosed.Load() {
			return
		}
		data := make([]byte, to.PieceBytes(index))
		if n, err := source.readResidentPiece(int(from.Offset/from.PieceLength)+index, 0, data, true); err != nil || n != len(data) {
			return
		}
		if !to.Verify(ctx, index, data) || cancelled() || source.reuseClosed.Load() || !c.reuseAllowed(int64(len(data))) {
			return
		}
		c.storage.mu.RLock()
		currentSource := c.storage.byHash[source.InfoHash] == source
		c.storage.mu.RUnlock()
		if !currentSource {
			return
		}
		c.mu.Lock()
		if c.reuseClosed.Load() || c.pieces == nil || c.reuseVerified[piece] || len(c.reuseVerified) >= 256 || (c.pieces[piece] != nil && c.pieces[piece].Complete()) {
			c.mu.Unlock()
			return
		}
		if c.reuseVerified == nil {
			c.reuseVerified = make(map[int]bool)
		}
		c.reuseVerified[piece] = true
		c.mu.Unlock()
		accepted, err := handle.TryReusePiece(piece, data)
		if !accepted || err != nil {
			c.mu.Lock()
			delete(c.reuseVerified, piece)
			c.mu.Unlock()
		}
	}()
	return true
}
