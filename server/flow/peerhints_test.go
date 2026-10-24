package flow

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPeerHintsExpiryNetworkAndBounds(t *testing.T) {
	now := time.Now()
	network := NetworkFingerprint([]string{"192.168.1.3", "2001:db8::1"})
	if network != NetworkFingerprint([]string{"2001:db8::1", "192.168.1.3"}) || NetworkFingerprint(nil) != "" {
		t.Fatal("unstable network fingerprint")
	}
	good := PeerHints{Version: 1, Network: network, SavedAt: now, Peers: []PeerHint{{IP: "127.0.0.1", Port: 51413}}}
	if !ValidPeerHints(good, network, now) {
		t.Fatal("valid hints rejected")
	}
	for _, mutate := range []func(*PeerHints){
		func(h *PeerHints) { h.Network = "other" }, func(h *PeerHints) { h.SavedAt = now.Add(-PeerHintsTTL - time.Second) }, func(h *PeerHints) { h.SavedAt = now.Add(time.Second) },
		func(h *PeerHints) { h.Peers = make([]PeerHint, 33) }, func(h *PeerHints) { h.Peers = []PeerHint{{IP: "169.254.169.254", Port: 80}} }, func(h *PeerHints) { h.Peers = []PeerHint{{IP: "239.1.1.1", Port: 80}} }, func(h *PeerHints) { h.Peers = []PeerHint{{IP: "1.1.1.1", Port: 0}} },
	} {
		invalid := good
		mutate(&invalid)
		if ValidPeerHints(invalid, network, now) {
			t.Fatalf("invalid hints accepted: %+v", invalid)
		}
	}
	name := filepath.Join(t.TempDir(), "hints.json")
	if err := WritePeerHints(name, good); err != nil {
		t.Fatal(err)
	}
	if len(ReadPeerHints(name, network, time.Now())) != 1 || len(ReadPeerHints(name, "other", time.Now())) != 0 {
		t.Fatal("restore policy failed")
	}
	if err := os.WriteFile(name, make([]byte, PeerHintsBytes+1), 0600); err != nil {
		t.Fatal(err)
	}
	if len(ReadPeerHints(name, network, time.Now())) != 0 {
		t.Fatal("oversized hints loaded")
	}
}

func TestPeerHintsRankRecentUsefulSuppliersAndDeduplicateAddresses(t *testing.T) {
	got := RankPeerHints([]PeerHint{
		{IP: "2001:0db8::1", Port: 42, UsefulRate: 2},
		{IP: "192.0.2.1", Port: 42, UsefulRate: 8, ServiceP95Ms: 500},
		{IP: "2001:db8::1", Port: 42, UsefulRate: 16, ServiceP95Ms: 100},
		{IP: "192.0.2.2", Port: 42, UsefulRate: 8, ServiceP95Ms: 100},
	})
	if len(got) != 3 || got[0].IP != "2001:db8::1" || got[1].IP != "192.0.2.2" {
		t.Fatalf("rank/dedup: %+v", got)
	}
	legacy := RankPeerHints([]PeerHint{{IP: "192.0.2.3", Port: 42}, {IP: "192.0.2.4", Port: 42}})
	if len(legacy) != 2 || legacy[0].IP != "192.0.2.3" {
		t.Fatal("legacy order changed")
	}
}
