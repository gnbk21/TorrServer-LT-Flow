package torr

import (
	"server/lt"
	"sync"
	"testing"
)

// Metadata/preload callers can retain a snapshot while cleanup detaches it.
// Exercise the actual Close/LTHandle paths under the race detector.
func TestCloseDetachesHandleWhileCallersSnapshot(t *testing.T) {
	for iteration := 0; iteration < 32; iteration++ {
		torrent := &Torrent{closeCh: make(chan struct{})}
		handle := &lt.Torrent{}
		torrent.lh.Store(handle)
		var callers sync.WaitGroup
		started := make(chan struct{}, 4)
		for worker := 0; worker < 4; worker++ {
			callers.Go(func() {
				started <- struct{}{}
				for i := 0; i < 1000; i++ {
					if got := torrent.LTHandle(); got != nil && got != handle {
						t.Error("published a different native identity")
					}
				}
			})
		}
		for worker := 0; worker < 4; worker++ {
			<-started
		}
		if !torrent.Close() || !torrent.Close() {
			t.Fatal("close must remain idempotent")
		}
		callers.Wait()
		if torrent.LTHandle() != nil {
			t.Fatal("closed torrent retained its published handle")
		}
	}
}
