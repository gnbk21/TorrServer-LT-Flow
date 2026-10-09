package torr

import (
	"context"
	"sync"
)

type preloadOperation struct {
	index  int
	ready  chan struct{}
	done   chan struct{}
	cancel context.CancelFunc
	once   sync.Once
}

// Preload returns at the buffer gate. A single owned worker retains the bounded
// handoff; same-file requests share readiness, a different file cancels and
// joins the previous owner before assigning reservations or priorities.
func (t *Torrent) Preload(ctx context.Context, index int, size int64, probe bool) {
	if t == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	for {
		t.preloadWorkMu.Lock()
		select {
		case <-t.closeCh:
			t.preloadWorkMu.Unlock()
			return
		default:
		}
		op := t.preloadWork
		if op != nil {
			select {
			case <-op.done:
				t.preloadWork = nil
				op = nil
			default:
			}
		}
		if op != nil && op.index != index {
			op.cancel()
			t.preloadWorkMu.Unlock()
			select {
			case <-op.done:
				continue
			case <-ctx.Done():
				return
			case <-t.closeCh:
				return
			}
		}
		if op == nil {
			workerCtx, cancel := context.WithCancel(ctx)
			op = &preloadOperation{index: index, ready: make(chan struct{}), done: make(chan struct{}), cancel: cancel}
			t.preloadWork = op
			go func() {
				defer close(op.done)
				defer cancel()
				signal := func() { op.once.Do(func() { close(op.ready) }) }
				defer signal()
				// Detached preloads must never turn a recoverable native/cache
				// error into an unhandled goroutine panic.
				defer func() {
					if recover() != nil {
						t.startupStage("FAILED")
					}
				}()
				t.fillPreload(workerCtx, index, size, probe, signal)
			}()
		}
		t.preloadWorkMu.Unlock()
		select {
		case <-op.ready:
			return
		case <-ctx.Done():
			return
		case <-t.closeCh:
			return
		}
	}
}

func (t *Torrent) stopPreload() {
	t.preloadWorkMu.Lock()
	op := t.preloadWork
	if op != nil {
		op.cancel()
	}
	t.preloadWorkMu.Unlock()
	if op != nil {
		<-op.done
	}
}
