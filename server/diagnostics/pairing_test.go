package diagnostics

import (
	"net/http/httptest"
	"testing"
	"testing/synctest"
	"time"
)

func TestPairingProofExpiryBoundAndIsolation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var s PairingStore
		p, e := s.Create("pc", time.Now())
		if e != nil {
			t.Fatal(e)
		}
		if s.Visit(p.ID, "phone", time.Now()) || s.Visit("invalid", "phone", time.Now()) {
			t.Fatal("lookup id became a proof")
		}
		if !s.Visit(p.Token, "pc", time.Now()) {
			t.Fatal("preview")
		}
		status, _ := s.Status(p.ID, time.Now())
		if status.DifferentClient || status.Token != "" {
			t.Fatal("preview claimed phone or leaked token")
		}
		s.Visit(p.Token, "phone", time.Now())
		r := httptest.NewRequest("GET", "/settings", nil)
		r.RemoteAddr = "phone:1234"
		r.Header.Set("Origin", "https://cf.lampa.mx")
		s.RecordAccess(r, "MANAGEMENT_ORIGIN_DENIED")
		r.RemoteAddr = "other:1234"
		s.RecordAccess(r, "AUTH_REQUIRED")
		status, _ = s.Status(p.ID, time.Now())
		if !status.DifferentClient || len(status.Evidence) != 1 || status.Evidence[0].Code != "MANAGEMENT_ORIGIN_DENIED" {
			t.Fatal(status)
		}
		for i := 1; i < PairingCapacity; i++ {
			if _, e = s.Create("pc", time.Now()); e != nil {
				t.Fatal(e)
			}
		}
		if _, e = s.Create("pc", time.Now()); e == nil {
			t.Fatal("unbounded storage")
		}
		time.Sleep(PairingLifetime)
		if s.Visit(p.Token, "phone", time.Now()) {
			t.Fatal("expired proof")
		}
		if _, ok := s.Status(p.ID, time.Now()); ok {
			t.Fatal("expired status")
		}
		if _, e = s.Create("pc", time.Now()); e != nil {
			t.Fatal("capacity never recovered")
		}
	})
}
