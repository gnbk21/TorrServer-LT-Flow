package flow

import (
	"context"
	"errors"
	"sync"
)

// FetchGroup coalesces only concurrent requests; it never caches credentials or
// results. Keys must include authorization and network policy identity.
type FetchGroup[T any] struct {
	mu      sync.Mutex
	entries map[string]*sharedFetch[T]
	running int
}
type sharedFetch[T any] struct {
	done   chan struct{}
	cancel context.CancelFunc
	owners int
	value  T
	err    error
}

var ErrFetchBusy = errors.New("remote fetch concurrency limit reached")

func (g *FetchGroup[T]) Do(ctx context.Context, key string, limit int, fetch func(context.Context) (T, error)) (T, error) {
	var zero T
	if err := ctx.Err(); err != nil {
		return zero, err
	}
	g.mu.Lock()
	if g.entries == nil {
		g.entries = make(map[string]*sharedFetch[T])
	}
	f := g.entries[key]
	if f == nil {
		if limit < 1 || g.running >= limit {
			g.mu.Unlock()
			return zero, ErrFetchBusy
		}
		sharedCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
		f = &sharedFetch[T]{done: make(chan struct{}), cancel: cancel}
		g.entries[key] = f
		g.running++
		go func() {
			defer cancel()
			value, err := fetch(sharedCtx)
			g.mu.Lock()
			f.value, f.err = value, err
			if g.entries[key] == f {
				delete(g.entries, key)
			}
			g.running--
			close(f.done)
			g.mu.Unlock()
		}()
	}
	f.owners++
	g.mu.Unlock()
	defer func() {
		g.mu.Lock()
		f.owners--
		if f.owners == 0 {
			if g.entries[key] == f {
				delete(g.entries, key)
			}
			f.cancel()
		}
		g.mu.Unlock()
	}()
	select {
	case <-ctx.Done():
		return zero, ctx.Err()
	case <-f.done:
		return f.value, f.err
	}
}
