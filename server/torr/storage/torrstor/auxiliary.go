package torrstor

import (
	"context"
	"errors"
	"io"
	"sync"
	"time"
)

var ErrAuxiliaryDeferred = errors.New("subtitle work deferred to protect playback or cache")

// AuxiliaryReader is an explicitly selected small sidecar. Its low-priority
// demand participates in the existing reconciler, never the playback windows,
// probe group, warm-session state, history, or consumption counters.
type AuxiliaryReader struct {
	cache       *Cache
	ctx         context.Context
	cancel      context.CancelFunc
	file        FileInfo
	first, last int
	offset      int64
	once        sync.Once
}

func (c *Cache) auxiliaryForegroundReady() bool {
	if c.auxiliaryClosed.Load() || c.backgroundLimited.Load() || c.flowWaiting.Load() > 0 || c.flowReconnect.Load() || time.Now().UnixMilli() < c.warmupRiskUntil.Load() {
		return false
	}
	c.preloadMu.Lock()
	preloading := len(c.preloadProtect) > 0
	c.preloadMu.Unlock()
	if preloading {
		return false
	}
	c.readersMu.Lock()
	defer c.readersMu.Unlock()
	for reader := range c.readers {
		if reader.internal {
			continue
		}
		for _, r := range [][2]int{reader.headReserve(), reader.tailReserve()} {
			for piece := r[0]; piece <= r[1]; piece++ {
				if !c.Have(piece) {
					return false
				}
			}
		}
	}
	return true
}

func (c *Cache) auxiliaryAllowed() bool {
	if !c.auxiliaryForegroundReady() {
		return false
	}
	if r := c.auxiliary.Load(); r != nil && r.ctx.Err() == nil && !c.auxiliaryBudgetFits(r.first, r.last) {
		return false
	}
	if c.storage != nil {
		c.storage.mu.RLock()
		defer c.storage.mu.RUnlock()
		for _, other := range c.storage.caches {
			if other != c && !other.auxiliaryForegroundReady() {
				return false
			}
		}
	}
	return true
}

func (c *Cache) auxiliaryPendingRange(first, last int) int64 {
	charge := int64(last-first+1) * c.PieceLength
	c.mu.RLock()
	defer c.mu.RUnlock()
	for p := first; p <= last; p++ {
		if piece := c.pieces[p]; piece != nil && !c.retainsLocked(p) {
			charge -= min(c.PieceLength, piece.SizeBytes())
		}
	}
	return max(0, charge)
}
func (c *Cache) auxiliaryPendingBytes() int64 {
	if r := c.auxiliary.Load(); r != nil && r.ctx.Err() == nil {
		return c.auxiliaryPendingRange(r.first, r.last)
	}
	return 0
}
func (c *Cache) auxiliaryBudgetFits(first, last int) bool {
	budget := globalCacheSize()
	if limit := c.resourceBudget.Load(); limit > 0 {
		budget = min(budget, limit)
	}
	return budget > 0 && c.Filled()+c.auxiliaryPendingRange(first, last) <= budget && c.streamingReserve() <= budget
}
func (c *Cache) NewAuxiliaryReader(ctx context.Context, file FileInfo) (*AuxiliaryReader, error) {
	if c == nil || file.Length <= 0 || file.Length > 4<<20 || file.Offset < 0 || c.PieceLength <= 0 || c.PieceLength > 32<<20 || file.Offset > int64(^uint64(0)>>1)-file.Length {
		return nil, ErrAuxiliaryDeferred
	}
	first, last := int(file.Offset/c.PieceLength), int((file.Offset+file.Length-1)/c.PieceLength)
	if first < 0 || last < first || last >= c.NumPieces || last-first >= 4096 {
		return nil, ErrAuxiliaryDeferred
	}
	charge := int64(last-first+1) * c.PieceLength
	if charge > 32<<20 || !c.auxiliaryAllowed() || !c.auxiliaryBudgetFits(first, last) {
		return nil, ErrAuxiliaryDeferred
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	r := &AuxiliaryReader{cache: c, ctx: ctx, cancel: cancel, file: file, first: first, last: last}
	if !c.auxiliary.CompareAndSwap(nil, r) {
		cancel()
		return nil, ErrAuxiliaryDeferred
	}
	context.AfterFunc(ctx, func() { _ = r.Close() })
	// Close/pressure may have raced the initial check before this owner became
	// visible to cancellation. Recheck after publication to close that gap.
	if !c.auxiliaryAllowed() {
		_ = r.Close()
		return nil, ErrAuxiliaryDeferred
	}
	c.lastApplyMs.Store(0)
	c.applyStreamPriorities()
	if r.ctx.Err() != nil {
		_ = r.Close()
		return nil, ErrAuxiliaryDeferred
	}
	return r, nil
}
func (r *AuxiliaryReader) Read(dst []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	if r.offset >= r.file.Length {
		return 0, io.EOF
	}
	if len(dst) == 0 {
		return 0, nil
	}
	if !r.cache.auxiliaryAllowed() {
		r.cancel()
		return 0, ErrAuxiliaryDeferred
	}
	position := r.file.Offset + r.offset
	piece, off := int(position/r.cache.PieceLength), position%r.cache.PieceLength
	if !r.cache.WaitForPiece(r.ctx, piece) {
		return 0, r.ctx.Err()
	}
	length := min(int64(len(dst)), r.file.Length-r.offset, r.cache.PieceLength-off)
	n, err := r.cache.readResidentPiece(piece, off, dst[:int(length)], true)
	r.offset += int64(n)
	if err == io.EOF && n > 0 {
		err = nil
	}
	return n, err
}
func (r *AuxiliaryReader) Close() error {
	r.once.Do(func() {
		r.cancel()
		r.cache.auxiliary.CompareAndSwap(r, nil)
		r.cache.lastApplyMs.Store(0)
		r.cache.applyStreamPriorities()
	})
	return nil
}
func (c *Cache) CancelAuxiliary() {
	if r := c.auxiliary.Load(); r != nil {
		r.cancel()
	}
}
func (c *Cache) auxiliaryReserved() int64 {
	if r := c.auxiliary.Load(); r != nil && r.ctx.Err() == nil {
		return int64(r.last-r.first+1) * c.PieceLength
	}
	return 0
}
func (c *Cache) auxiliaryProtection() [][2]int {
	if r := c.auxiliary.Load(); r != nil && r.ctx.Err() == nil {
		return [][2]int{{r.first, r.last}}
	}
	return nil
}
func (c *Cache) auxiliaryDemand(blocked bool) []int {
	r := c.auxiliary.Load()
	if r == nil || r.ctx.Err() != nil {
		return nil
	}
	if blocked || !c.auxiliaryAllowed() {
		r.cancel()
		return nil
	}
	pieces := make([]int, 0, r.last-r.first+1)
	for p := r.first; p <= r.last; p++ {
		if !c.Have(p) {
			pieces = append(pieces, p)
		}
	}
	return pieces
}
