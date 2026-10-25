package flow

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"sort"
	"strings"
	"time"
)

const PeerHintsTTL = 10 * time.Minute
const PeerHintsBytes = 8192

type PeerHint struct {
	IP           string `json:"ip"`
	Port         int    `json:"port"`
	UsefulRate   int64  `json:"useful_rate,omitempty"`
	ServiceP95Ms int64  `json:"service_p95_ms,omitempty"`
}
type PeerHints struct {
	Version int        `json:"version"`
	Network string     `json:"network"`
	SavedAt time.Time  `json:"saved_at"`
	Peers   []PeerHint `json:"peers"`
}

func NetworkFingerprint(addresses []string) string {
	if len(addresses) == 0 {
		return ""
	}
	copyAddresses := append([]string(nil), addresses...)
	sort.Strings(copyAddresses)
	value := sha256.Sum256([]byte(strings.Join(copyAddresses, "\n")))
	return hex.EncodeToString(value[:])
}

func ValidPeerHints(h PeerHints, network string, now time.Time) bool {
	if h.Version != 1 || network == "" || h.Network != network || now.Before(h.SavedAt) || now.Sub(h.SavedAt) > PeerHintsTTL || len(h.Peers) > 32 {
		return false
	}
	for _, p := range h.Peers {
		ip := net.ParseIP(p.IP)
		if ip == nil || ip.IsUnspecified() || ip.IsMulticast() || ip.IsLinkLocalUnicast() || p.Port < 1 || p.Port > 65535 ||
			p.UsefulRate < 0 || p.UsefulRate > 1<<40 || p.ServiceP95Ms < 0 || p.ServiceP95Ms > 300000 {
			return false
		}
	}
	return true
}

func ReadPeerHints(name, network string, now time.Time) []PeerHint {
	st, err := os.Lstat(name)
	if err != nil || !st.Mode().IsRegular() || st.Size() > PeerHintsBytes {
		return nil
	}
	data, err := ReadDHTFile(name)
	if err != nil || len(data) > PeerHintsBytes {
		return nil
	}
	var hints PeerHints
	if json.Unmarshal(data, &hints) != nil || !ValidPeerHints(hints, network, now) {
		return nil
	}
	return RankPeerHints(hints.Peers)
}

// RankPeerHints preserves legacy entries without scores and canonicalizes
// duplicates. Scores only describe recent observations on the saved network.
func RankPeerHints(peers []PeerHint) []PeerHint {
	out := make([]PeerHint, 0, min(32, len(peers)))
	seen := make(map[string]int, 32)
	for _, peer := range peers {
		ip := net.ParseIP(peer.IP)
		if ip == nil || peer.Port < 1 || peer.Port > 65535 {
			continue
		}
		peer.IP = ip.String()
		key := net.JoinHostPort(peer.IP, fmt.Sprint(peer.Port))
		if i, ok := seen[key]; ok {
			if peer.UsefulRate > out[i].UsefulRate {
				out[i] = peer
			}
			continue
		}
		if len(out) == 32 {
			break
		}
		seen[key] = len(out)
		out = append(out, peer)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].UsefulRate != out[j].UsefulRate {
			return out[i].UsefulRate > out[j].UsefulRate
		}
		a, b := out[i].ServiceP95Ms, out[j].ServiceP95Ms
		if a == 0 {
			a = 300001
		}
		if b == 0 {
			b = 300001
		}
		return a < b
	})
	return out
}

func WritePeerHints(name string, h PeerHints) error {
	if !ValidPeerHints(h, h.Network, time.Now()) {
		return errors.New("invalid peer hints")
	}
	data, err := json.Marshal(h)
	if err != nil || len(data) > PeerHintsBytes {
		return errors.New("peer hints exceed bound")
	}
	return WriteDHTFile(name, data)
}
