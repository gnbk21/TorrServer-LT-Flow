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
	claim := PlaybackClaim{Hash: strings.Repeat("a", 40), Index: 7, Expires: 1060}
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

func TestSubtitleCapabilityIsAdditiveAndV2Scoped(t *testing.T) {
	s, _ := NewCapabilitySigner()
	now := time.Now()
	for _, hash := range []string{strings.Repeat("a", 40), strings.Repeat("b", 64)} {
		old := PlaybackClaim{Hash: hash, Index: 1, Expires: now.Add(time.Minute).Unix()}
		token, err := s.Mint(old, now)
		if err != nil {
			t.Fatal(err)
		}
		got, err := s.Verify(token, now)
		if err != nil || got.SubtitleIndex != 0 {
			t.Fatal("old authority expanded", got, err)
		}
		old.SubtitleIndex = 2
		token, err = s.Mint(old, now)
		if err != nil {
			t.Fatal(err)
		}
		got, err = s.Verify(token, now)
		if err != nil || got != old {
			t.Fatal("selected scope lost", got, err)
		}
		for _, bad := range []int{-1, 1} {
			old.SubtitleIndex = bad
			if _, err = s.Mint(old, now); err == nil {
				t.Fatal("invalid subtitle scope", bad)
			}
		}
	}
}
