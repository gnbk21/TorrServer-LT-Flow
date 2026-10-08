package torrstor

import (
	"errors"
	"os"
	"path/filepath"
	"server/lt"
)

// Ranges include whole boundary pieces, charged once in the job quota. Paused
// jobs retain bytes without requesting them. No second media store is created.
type PreparationRange struct {
	First, Last int
	Active      bool
}
type PreparationStorage struct {
	Root   string
	Ranges map[string]PreparationRange
}

func clonePreparationRanges(in map[string]PreparationRange) map[string]PreparationRange {
	out := make(map[string]PreparationRange, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func (s *Storage) ConfigurePreparation(hash [20]byte, plan PreparationStorage) error {
	s.mu.Lock()
	if s.preparations == nil {
		s.preparations = make(map[[20]byte]PreparationStorage)
	}
	if len(plan.Ranges) == 0 {
		delete(s.preparations, hash)
	} else {
		s.preparations[hash] = PreparationStorage{plan.Root, clonePreparationRanges(plan.Ranges)}
	}
	c := s.byHash[hash]
	s.mu.Unlock()
	if c != nil {
		return c.configurePreparation(plan)
	}
	return nil
}

func (s *Storage) PreparationPlan(hash [20]byte) (PreparationStorage, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.preparations[hash]
	return PreparationStorage{p.Root, clonePreparationRanges(p.Ranges)}, ok && len(p.Ranges) > 0
}

// Migration holds the existing piece lock, so native reads/writes cannot see a
// half-copied backend. After success the RAM slice is released immediately.
func (c *Cache) configurePreparation(plan PreparationStorage) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.diskRoot != "" && c.diskRoot != plan.Root {
		return errors.New("preparation storage path changed")
	}
	if err := os.MkdirAll(filepath.Join(plan.Root, hashHex(c.InfoHash)), 0700); err != nil {
		return err
	}
	var migrationBuffer []byte
	for _, p := range c.pieces {
		p.mu.Lock()
		if p.mem != nil {
			dp := newDiskPiece(p, plan.Root)
			if p.size > 0 {
				if n, err := dp.WriteAt(p.mem.buf[:p.size], 0); err != nil || int64(n) != p.size {
					dp.Close()
					p.mu.Unlock()
					return errors.New("cannot persist preparation piece")
				}
			}
			p.mem.Release()
			p.mem = nil
			p.disk = dp
		} else if p.disk != nil && p.disk.dir != filepath.Join(plan.Root, hashHex(c.InfoHash)) {
			// A retained manager root may differ from a new ordinary disk-cache
			// path. Migrate its existing backend too, using bounded scratch space.
			dp := newDiskPiece(p, plan.Root)
			// Relative paths, case aliases and user-selected directory symlinks
			// may identify the same file. Never copy then unlink that backend.
			oldInfo, oldErr := os.Stat(p.disk.name)
			newInfo, newErr := os.Stat(dp.name)
			if oldErr == nil && newErr == nil && os.SameFile(oldInfo, newInfo) {
				p.disk.Close()
				p.disk = dp
				p.mu.Unlock()
				continue
			}
			if migrationBuffer == nil {
				migrationBuffer = make([]byte, 64<<10)
			}
			buffer := migrationBuffer
			for off := int64(0); off < p.size; {
				n, err := p.disk.ReadAt(buffer[:min(int64(len(buffer)), p.size-off)], off)
				if err != nil || n == 0 {
					dp.Close()
					p.mu.Unlock()
					return errors.New("cannot read preparation migration piece")
				}
				if written, err := dp.WriteAt(buffer[:n], off); err != nil || written != n {
					dp.Close()
					p.mu.Unlock()
					return errors.New("cannot persist preparation migration piece")
				}
				off += int64(n)
			}
			if p.size > 0 {
				if err := dp.Sync(); err != nil {
					dp.Close()
					p.mu.Unlock()
					return errors.New("cannot sync preparation migration piece")
				}
			}
			p.disk.Release()
			p.disk = dp
		}
		p.mu.Unlock()
	}
	c.diskRoot = plan.Root
	c.preparation = clonePreparationRanges(plan.Ranges)
	return nil
}

func (c *Cache) retainsLocked(piece int) bool {
	for _, r := range c.preparation {
		if piece >= r.First && piece <= r.Last {
			return true
		}
	}
	return false
}
func (c *Cache) preparationRetains(piece int) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.retainsLocked(piece)
}

// Bound ordinary requests to 32 missing pieces per job, preserving throughput
// without giving an entire episode time-critical deadlines.
func (c *Cache) preparationDemand() []int {
	if c.backgroundLimited.Load() {
		return nil
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	var out []int
	for _, r := range c.preparation {
		if !r.Active {
			continue
		}
		n := 0
		for i := r.First; i <= r.Last && n < 32; i++ {
			if p := c.pieces[i]; p != nil && p.Complete() {
				continue
			}
			out = append(out, i)
			n++
		}
	}
	return out
}

func (c *Cache) TickPreparation(handle *lt.Torrent) {
	if handle == nil {
		return
	}
	c.handle.Store(handle)
	c.applyStreamPriorities()
}

func (c *Cache) PreparationError() bool { return c.preparationWriteError.Swap(false) }
func (c *Cache) BeginPreparationCleanup() bool {
	c.readersMu.Lock()
	defer c.readersMu.Unlock()
	if len(c.readers) > 0 {
		return false
	}
	c.preparationCleaning = true
	return true
}

func (s *Storage) ResetPreparationPlans() {
	s.mu.Lock()
	s.preparations = make(map[[20]byte]PreparationStorage)
	s.mu.Unlock()
}

// Progress counts only native hash-verified pieces and intersects boundary
// pieces with the selected file. Responsive partial blocks never establish ready.
func (c *Cache) PreparationProgress(offset, length int64) (verified, contiguous int64) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if length <= 0 || c.PieceLength <= 0 {
		return
	}
	end := offset + length
	prefix := true
	for i := int(offset / c.PieceLength); int64(i)*c.PieceLength < end; i++ {
		p := c.pieces[i]
		bytes := min(end, int64(i+1)*c.PieceLength) - max(offset, int64(i)*c.PieceLength)
		if p != nil && p.Complete() {
			verified += bytes
			if prefix {
				contiguous += bytes
			}
		} else {
			prefix = false
		}
	}
	return
}

// Flush retained pieces before publishing READY. Restart still hashes them:
// filesystem durability alone does not establish torrent integrity.
func (c *Cache) FlushPreparation(first, last int) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for i := first; i <= last; i++ {
		if p := c.pieces[i]; p != nil {
			p.mu.RLock()
			if p.disk != nil && p.complete {
				if err := p.disk.Sync(); err != nil {
					p.mu.RUnlock()
					return err
				}
			}
			p.mu.RUnlock()
		}
	}
	return nil
}
