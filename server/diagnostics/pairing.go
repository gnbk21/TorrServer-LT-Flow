package diagnostics

import (
	"crypto/rand"
	"errors"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"
)

const PairingLifetime = 5 * time.Minute
const PairingCapacity = 64

type AccessEvidence struct {
	At     time.Time `json:"at"`
	Code   string    `json:"code"`
	Origin string    `json:"origin,omitempty"`
	peer   string
}
type PairingProof struct {
	ID               string           `json:"id"`
	Token            string           `json:"token,omitempty"`
	ExpiresAt        time.Time        `json:"expires_at"`
	ReachedAt        *time.Time       `json:"reached_at,omitempty"`
	DifferentClient  bool             `json:"different_client"`
	Evidence         []AccessEvidence `json:"evidence"`
	creator, visitor string
}
type PairingStore struct {
	mu       sync.Mutex
	proofs   map[string]*PairingProof
	evidence []AccessEvidence
}

var Pairing PairingStore

func requestPeer(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	return host // Never trust a caller-controlled forwarding header here.
}
func (s *PairingStore) prune(now time.Time) {
	for id, p := range s.proofs {
		if !now.Before(p.ExpiresAt) {
			delete(s.proofs, id)
		}
	}
	kept := s.evidence[:0]
	for _, e := range s.evidence {
		if now.Sub(e.At) < PairingLifetime {
			kept = append(kept, e)
		}
	}
	clear(s.evidence[len(kept):])
	s.evidence = kept
}
func (s *PairingStore) Create(peer string, now time.Time) (PairingProof, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.prune(now)
	if len(s.proofs) >= PairingCapacity {
		return PairingProof{}, errors.New("pairing proof capacity reached")
	}
	if s.proofs == nil {
		s.proofs = map[string]*PairingProof{}
	}
	p := PairingProof{ID: rand.Text(), Token: rand.Text(), ExpiresAt: now.Add(PairingLifetime), creator: peer, Evidence: []AccessEvidence{}}
	s.proofs[p.ID] = &p
	return p, nil
}
func (s *PairingStore) Visit(token, peer string, now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.prune(now)
	for _, p := range s.proofs {
		if p.Token != token {
			continue
		}
		// A local preview must not prevent a subsequent phone request proving
		// a different address. Neither observation proves a physical device type.
		if p.ReachedAt == nil || (!p.DifferentClient && peer != p.creator) {
			at := now
			p.ReachedAt = &at
			p.visitor = peer
			p.DifferentClient = peer != p.creator
		}
		return true
	}
	return false
}
func (s *PairingStore) Status(id string, now time.Time) (PairingProof, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.prune(now)
	p, ok := s.proofs[id]
	if !ok {
		return PairingProof{}, false
	}
	copy := *p
	copy.Token = ""
	copy.Evidence = []AccessEvidence{}
	if p.ReachedAt != nil {
		at := *p.ReachedAt
		copy.ReachedAt = &at
	}
	for _, e := range s.evidence {
		if p.visitor != "" && e.peer == p.visitor && !e.At.Before(p.ExpiresAt.Add(-PairingLifetime)) {
			copy.Evidence = append(copy.Evidence, e)
		}
	}
	return copy, true
}
func (s *PairingStore) RecordAccess(r *http.Request, code string) {
	if code != "AUTH_REQUIRED" && code != "AUTH_ACCEPTED" && code != "MANAGEMENT_ORIGIN_DENIED" && code != "MANAGEMENT_HOST_DENIED" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	s.prune(now)
	if len(s.proofs) == 0 {
		return
	}
	origin := ""
	if raw := r.Header.Get("Origin"); len(raw) <= 512 {
		if u, e := url.Parse(raw); e == nil && u.User == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != "" {
			origin = u.Scheme + "://" + u.Host
		}
	}
	e := AccessEvidence{At: now, Code: code, Origin: origin, peer: requestPeer(r)}
	if len(s.evidence) >= 128 {
		copy(s.evidence, s.evidence[1:])
		s.evidence[len(s.evidence)-1] = e
	} else {
		s.evidence = append(s.evidence, e)
	}
}

func PairingRequestPeer(r *http.Request) string { return requestPeer(r) }
