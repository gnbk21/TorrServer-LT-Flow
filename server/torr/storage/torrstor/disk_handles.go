package torrstor

import (
	"container/list"
	"os"
	"path/filepath"
	"sync"
)

// diskHandles is shared by a Storage, not by every piece. References cover the
// actual syscall; eviction never closes an in-use handle. Waiting for a slot
// bounds descriptors even when many HTTP readers and native workers overlap.
type diskHandles struct {
	mu    sync.Mutex
	cond  *sync.Cond
	limit int
	files map[string]*diskHandle
	lru   list.List
}

type diskHandle struct {
	file     *os.File
	writable bool
	refs     int
	element  *list.Element
}

func newDiskHandles(limit int) *diskHandles {
	p := &diskHandles{limit: max(1, limit), files: make(map[string]*diskHandle)}
	p.cond = sync.NewCond(&p.mu)
	return p
}

func (p *diskHandles) acquire(name string, write bool) (*os.File, func(), error) {
	return p.acquireMode(name, write, write)
}

func (p *diskHandles) acquireMode(name string, write, create bool) (*os.File, func(), error) {
	p.mu.Lock()
	for {
		if h := p.files[name]; h != nil {
			if write && !h.writable {
				if h.refs > 0 {
					p.cond.Wait()
					continue
				}
				p.removeLocked(name, h)
			} else {
				h.refs++
				p.lru.MoveToBack(h.element)
				p.mu.Unlock()
				return h.file, func() { p.release(h) }, nil
			}
		}
		if len(p.files) >= p.limit {
			var victim *list.Element
			for e := p.lru.Front(); e != nil; e = e.Next() {
				if p.files[e.Value.(string)].refs == 0 {
					victim = e
					break
				}
			}
			if victim == nil {
				p.cond.Wait()
				continue
			}
			key := victim.Value.(string)
			p.removeLocked(key, p.files[key])
		}
		flags := os.O_RDONLY
		if write {
			flags = os.O_RDWR
			if create {
				if err := os.MkdirAll(filepath.Dir(name), 0o777); err != nil {
					p.mu.Unlock()
					return nil, nil, err
				}
				flags |= os.O_CREATE
			}
		}
		f, err := os.OpenFile(name, flags, 0o666)
		if err != nil {
			p.mu.Unlock()
			return nil, nil, err
		}
		h := &diskHandle{file: f, writable: write, refs: 1, element: p.lru.PushBack(name)}
		p.files[name] = h
		p.mu.Unlock()
		return f, func() { p.release(h) }, nil
	}
}

func (p *diskHandles) release(h *diskHandle) {
	p.mu.Lock()
	h.refs--
	p.cond.Broadcast()
	p.mu.Unlock()
}

func (p *diskHandles) removeLocked(name string, h *diskHandle) {
	_ = h.file.Close()
	p.lru.Remove(h.element)
	delete(p.files, name)
}

// Caller fences operations on this piece before closing/removing its file.
func (p *diskHandles) closePath(name string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for {
		h := p.files[name]
		if h == nil {
			return
		}
		if h.refs > 0 {
			p.cond.Wait()
			continue
		}
		p.removeLocked(name, h)
		p.cond.Broadcast()
		return
	}
}
