package flow

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"os"
	"sort"
	"strings"
	"time"
)

const PeerHintsTTL = 10 * time.Minute
const PeerHintsBytes = 8192

type PeerHint struct {
	IP   string `json:"ip"`
	Port int    `json:"port"`
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
		if ip == nil || ip.IsUnspecified() || ip.IsMulticast() || ip.IsLinkLocalUnicast() || p.Port < 1 || p.Port > 65535 {
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
	return hints.Peers
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
