package flow

import "sync"

// RequestGate atomically blocks new work only after every accepted request
// has finished. It does not interrupt an active player or wait indefinitely.
type RequestGate struct {
	mu      sync.Mutex
	active  int
	enabled bool
}

var Maintenance RequestGate

func (g *RequestGate) Enter() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.enabled {
		return false
	}
	g.active++
	return true
}
func (g *RequestGate) Leave() {
	g.mu.Lock()
	if g.active > 0 {
		g.active--
	}
	g.mu.Unlock()
}
func (g *RequestGate) Set(enabled bool) (bool, int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if !enabled || g.active == 0 {
		g.enabled = enabled
	}
	return g.enabled, g.active
}
