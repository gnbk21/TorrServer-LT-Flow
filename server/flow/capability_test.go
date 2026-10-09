package flow

import (
	"strings"
	"testing"
	"time"
)

func TestPlaybackCapabilityScopeTamperingAndExpiry(t *testing.T) {
	s, err := NewCapabilitySigner()
	if err != nil {
		t.Fatal(err)
	}
	other, _ := NewCapabilitySigner()
	now := time.Unix(1000, 0)
	claim := PlaybackClaim{strings.Repeat("a", 40), 7, 1060}
	token, err := s.Mint(claim, now)
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.Verify(token, now.Add(59*time.Second))
	if err != nil || got != claim {
		t.Fatal(got, err)
	}
	for _, candidate := range []string{token + "x", "x" + token, strings.Repeat("a", 513)} {
		if _, err := s.Verify(candidate, now); err == nil {
			t.Fatal("accepted tampering")
		}
	}
	if _, err := other.Verify(token, now); err == nil {
		t.Fatal("capability survived signer rotation")
	}
	if _, err := s.Verify(token, now.Add(time.Minute)); err == nil {
		t.Fatal("accepted expired link")
	}
	claim.Index = 0
	if _, err := s.Mint(claim, now); err == nil {
		t.Fatal("accepted unscoped file")
	}
}
func TestManagementLimiterBoundsAndRefill(t *testing.T) {
	var l ManagementLimiter
	now := time.Unix(1000, 0)
	if !l.Allow("one", 2, now) || !l.Allow("one", 2, now) || l.Allow("one", 2, now) {
		t.Fatal("incorrect burst")
	}
	if !l.Allow("one", 2, now.Add(30*time.Second)) || l.Allow("one", 2, now.Add(30*time.Second)) {
		t.Fatal("incorrect refill")
	}
	for i := 0; i < 5000; i++ {
		l.Allow(string(rune(i))+"new", 1, now)
	}
	if len(l.entries) > 4096 {
		t.Fatal("unbounded client map")
	}
	if !l.Allow("after-expiry", 1, now.Add(3*time.Minute)) {
		t.Fatal("expired clients were not reclaimed")
	}
}
