package flow

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// RequestGate atomically blocks new work only after every accepted request
// has finished. It does not interrupt an active player or wait indefinitely.
type RequestGate struct {
	mu      sync.Mutex
	active  int
	token   string
	expires time.Time
}

var Maintenance RequestGate

func (g *RequestGate) Enter() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.expire()
	if g.token != "" {
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

// Acquire is exclusive: a backup cannot release an updater's gate. A bounded
// updater lease reopens playback if its client crashes before shutdown.
func (g *RequestGate) Acquire(ttl time.Duration) (string, int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.expire()
	if g.token != "" || g.active != 0 {
		return "", g.active
	}
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return "", g.active
	}
	g.token = hex.EncodeToString(nonce[:])
	if ttl > 0 {
		g.expires = time.Now().Add(ttl)
	}
	return g.token, g.active
}
func (g *RequestGate) Release(token string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.expire()
	if token == "" || g.token != token {
		return false
	}
	g.token, g.expires = "", time.Time{}
	return true
}
func (g *RequestGate) expire() {
	if !g.expires.IsZero() && !time.Now().Before(g.expires) {
		g.token, g.expires = "", time.Time{}
	}
}
