package torrstor

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestDiskHandlesReuseBoundAndUpgrade(t *testing.T) {
	p := newDiskHandles(2)
	root := t.TempDir()
	a := filepath.Join(root, "a")
	f, done, err := p.acquire(a, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.WriteAt([]byte("test"), 0); err != nil {
		t.Fatal(err)
	}
	done()
	f2, done, err := p.acquire(a, false)
	if err != nil || f2 != f {
		t.Fatal("handle not reused", err)
	}
	done()
	for _, name := range []string{"b", "c"} {
		_, done, err := p.acquire(filepath.Join(root, name), true)
		if err != nil {
			t.Fatal(err)
		}
		done()
	}
	if len(p.files) != 2 {
		t.Fatal("descriptor limit exceeded")
	}
	if _, err := f.Stat(); err == nil {
		t.Fatal("LRU handle not closed")
	}
	_, done, err = p.acquire(a, false)
	if err != nil {
		t.Fatal(err)
	}
	done()
	f, done, err = p.acquire(a, true)
	if err != nil {
		t.Fatal("read-only handle upgrade", err)
	}
	_, err = f.WriteAt([]byte("new!"), 0)
	done()
	if err != nil {
		t.Fatal(err)
	}
	for name := range p.files {
		p.closePath(name)
	}
	if len(p.files) != 0 {
		t.Fatal("handles leaked")
	}
}

func TestDiskHandlesWaitAndCloseBeforeDelete(t *testing.T) {
	p := newDiskHandles(1)
	root := t.TempDir()
	a, b := filepath.Join(root, "a"), filepath.Join(root, "b")
	_, release, err := p.acquire(a, true)
	if err != nil {
		t.Fatal(err)
	}
	acquired := make(chan error, 1)
	go func() {
		_, done, err := p.acquire(b, true)
		if err == nil {
			done()
		}
		acquired <- err
	}()
	select {
	case <-acquired:
		t.Fatal("in-use handle evicted")
	case <-time.After(20 * time.Millisecond):
	}
	release()
	select {
	case err := <-acquired:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("pool waiter stuck")
	}
	p.closePath(b)
	if err := os.Remove(b); err != nil {
		t.Fatal("close did not permit deletion", err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for n := 0; n < 100; n++ {
				_, done, err := p.acquire(a, false)
				if err != nil {
					t.Error(err)
					return
				}
				done()
			}
		}()
	}
	wg.Wait()
	p.closePath(a)
}
