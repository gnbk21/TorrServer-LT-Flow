// Package torrstor is the Go-side piece cache backing libtorrent's custom
// disk_interface (defined in server/lt). It owns one Cache per running
// torrent and routes the disk callbacks emitted by libtorrent's disk
// threads to the right per-torrent Cache.
//
// Etap 4.1 — in-memory only: every Piece's bytes live in a MemPiece.
// Etap 4.2 — adds DiskPiece (TorrentsSavePath/<hash>/<id>) plus the resume
// scan that populates the have-bitmap from existing piece files.
package torrstor

import (
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"server/lt"
)

// Storage is the process-wide registry of per-torrent caches. Plug it
// into libtorrent via Install(); calls from libtorrent's disk threads
// land on its Read/Write/Open/Close/Deleted/Have methods.
type Storage struct {
	mu                sync.RWMutex
	caches            map[int64]*Cache    // by libtorrent storage_id
	byHash            map[[20]byte]*Cache // by info hash for Reader lookup from torr
	resumes           map[[20]byte]verifiedResume
	preparations      map[[20]byte]PreparationStorage
	verifiedReads     map[[20]byte]bool
	networkRecovering atomic.Bool
}

type verifiedResume struct {
	bitmap    []byte
	totalSize int64
}

// SetVerifiedResume is called before native addition. The consumed state shares
// the same verified have-bitmap with native and Go readers; it is never inferred
// from file size in a disk callback.
func (s *Storage) SetVerifiedResume(hash [20]byte, bitmap []byte, totalSize int64) {
	s.mu.Lock()
	s.resumes[hash] = verifiedResume{append([]byte(nil), bitmap...), totalSize}
	s.mu.Unlock()
}

func (s *Storage) ClearVerifiedResume(hash [20]byte) {
	s.mu.Lock()
	delete(s.resumes, hash)
	delete(s.verifiedReads, hash)
	s.mu.Unlock()
}

// Mirror bytes must pass native piece hashes before external readers see them.
// This is sticky for the handle lifetime, including in-flight removed mirrors.
func (s *Storage) RequireVerifiedReads(hash [20]byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.verifiedReads == nil {
		s.verifiedReads = make(map[[20]byte]bool)
	}
	s.verifiedReads[hash] = true
	if c := s.byHash[hash]; c != nil {
		c.verifiedReads.Store(true)
	}
}

// NewStorage constructs an empty registry.
func NewStorage() *Storage {
	return &Storage{
		caches:       map[int64]*Cache{},
		byHash:       map[[20]byte]*Cache{},
		resumes:      map[[20]byte]verifiedResume{},
		preparations: map[[20]byte]PreparationStorage{},
	}
}

var (
	globalOnce sync.Once
	globalSt   *Storage
)

// Global returns the process-wide Storage singleton (constructed on first
// access). BTServer.Connect plumbs it into libtorrent via Install().
func Global() *Storage {
	globalOnce.Do(func() { globalSt = NewStorage() })
	return globalSt
}

// Install registers s as the disk_io backend for the next session_new.
// Already-running sessions keep their original disk_io.
func (s *Storage) Install() error {
	return lt.RegisterStorageCallbacks(lt.StorageCallbacks{
		Open:       s.callbackOpen,
		Close:      s.callbackClose,
		Deleted:    s.callbackDeleted,
		Read:       s.callbackRead,
		Write:      s.callbackWrite,
		Have:       s.callbackHave,
		Prune:      s.callbackPrune,
		Evict:      s.callbackEvict,
		Size:       s.callbackSize,
		ClearPiece: s.callbackClearPiece,
	})
}

// Uninstall reverts to libtorrent's default disk_io on the next session_new.
func (s *Storage) Uninstall() error {
	return lt.RegisterStorageCallbacks(lt.StorageCallbacks{})
}

// CacheByHash returns the live Cache for the given v1 info hash, or nil
// if no torrent with that hash is currently registered.
func (s *Storage) CacheByHash(hash [20]byte) *Cache {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.byHash[hash]
}

type AllocationStatus struct {
	Caches                 int   `json:"caches"`
	ActiveCaches           int   `json:"active_caches"`
	IdleCaches             int   `json:"idle_caches"`
	WarmCaches             int   `json:"warm_caches"`
	WarmResidentBytes      int64 `json:"warm_resident_bytes"`
	ResidentBytes          int64 `json:"resident_bytes"`
	ActiveResidentBytes    int64 `json:"active_resident_bytes"`
	IdleResidentBytes      int64 `json:"idle_resident_bytes"`
	EffectiveCapacityBytes int64 `json:"effective_capacity_bytes"`
	ProtectedBytes         int64 `json:"protected_bytes"`
	ActiveReaders          int   `json:"active_readers"`
}

// Count each cache once, regardless of how many sessions share it. Capacities
// are eviction budgets; they are not process RSS or a global RAM hard limit.
func (s *Storage) Allocations() AllocationStatus {
	s.mu.RLock()
	caches := make([]*Cache, 0, len(s.caches))
	for _, c := range s.caches {
		caches = append(caches, c)
	}
	s.mu.RUnlock()
	var out AllocationStatus
	for _, c := range caches {
		filled := c.Filled()
		readers := c.StreamingReaders()
		out.Caches++
		out.ResidentBytes += filled
		out.EffectiveCapacityBytes += c.capacity()
		out.ProtectedBytes += c.streamingReserve()
		out.ActiveReaders += readers
		if readers > 0 {
			out.ActiveCaches++
			out.ActiveResidentBytes += filled
		} else {
			out.IdleCaches++
			out.IdleResidentBytes += filled
			c.preloadMu.Lock()
			warm := time.Now().Before(c.warmUntil)
			c.preloadMu.Unlock()
			if warm {
				out.WarmCaches++
				out.WarmResidentBytes += filled
			}
		}
	}
	return out
}

// ----- lt.StorageCallbacks dispatch -----

func (s *Storage) callbackOpen(storage int64, hash [20]byte, numPieces int, pieceLength int64) {
	c := newCache(s, storage, hash, numPieces, pieceLength)
	s.mu.Lock()
	resume := s.resumes[hash]
	delete(s.resumes, hash)
	c.resume = resume.bitmap
	c.totalSize.Store(resume.totalSize)
	c.verifiedReads.Store(s.verifiedReads[hash])
	if plan, ok := s.preparations[hash]; ok {
		c.diskRoot = plan.Root
		c.preparation = clonePreparationRanges(plan.Ranges)
	}
	c.scanLocalPieces()
	s.caches[storage] = c
	s.byHash[hash] = c
	s.mu.Unlock()
	// In UseDisk mode, eagerly materialise any pre-existing piece
	// files so Have()/Reader hit them without going through the lazy
	// reconstruction path in readPiece.
}

func (s *Storage) callbackClose(storage int64) {
	s.mu.Lock()
	c := s.caches[storage]
	if c != nil {
		// Disk callbacks can finish after the same hash was re-added with a
		// different storage ID. Only the current owner may remove its lookup
		// and sticky mirror gate.
		if s.byHash[c.InfoHash] == c {
			delete(s.byHash, c.InfoHash)
			delete(s.verifiedReads, c.InfoHash)
		}
		delete(s.caches, storage)
	}
	s.mu.Unlock()
	if c != nil {
		c.close()
	}
}

func (s *Storage) callbackDeleted(storage int64) {
	s.mu.RLock()
	c := s.caches[storage]
	s.mu.RUnlock()
	if c != nil {
		c.wipe()
	}
}

func (s *Storage) callbackRead(storage int64, piece int, offset int64, dst []byte) (int, error) {
	c := s.lookup(storage)
	if c == nil {
		return 0, errStorageMissing
	}
	return c.readPiece(piece, offset, dst)
}

func (s *Storage) callbackWrite(storage int64, piece int, offset int64, src []byte) (int, error) {
	c := s.lookup(storage)
	if c == nil {
		return 0, errStorageMissing
	}
	return c.writePiece(piece, offset, src)
}

func (s *Storage) callbackHave(storage int64, piece int) bool {
	c := s.lookup(storage)
	if c == nil {
		return false
	}
	return c.Have(piece)
}

func (s *Storage) callbackPrune(storage int64, piece int) bool {
	c := s.lookup(storage)
	return c != nil && c.prunePartial(piece)
}

func (s *Storage) callbackEvict(storage int64, piece int) bool {
	c := s.lookup(storage)
	return c != nil && c.evictComplete(piece)
}

func (s *Storage) callbackSize(storage int64, totalSize int64) {
	if c := s.lookup(storage); c != nil {
		c.totalSize.Store(totalSize)
	}
}

func (s *Storage) callbackClearPiece(storage int64, piece int) {
	if c := s.lookup(storage); c != nil {
		c.InvalidatePiece(piece)
	}
}

func (s *Storage) lookup(storage int64) *Cache {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.caches[storage]
}

// ----- errors -----

var (
	errStorageMissing = errors.New("torrstor: no cache for storage_id")
	errOutOfPiece     = errors.New("torrstor: offset beyond piece length")
	_                 = io.EOF // keep import even when only used indirectly
)
