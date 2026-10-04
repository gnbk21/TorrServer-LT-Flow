package torr

import (
	"math/rand/v2"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"server/flow"
	"server/lt"
	"server/settings"
)

var peerHintsDiskMu sync.Mutex

func peerHintsNetwork() string {
	addresses, err := localNetworkAddresses()
	if err != nil {
		return ""
	}
	return flow.NetworkFingerprint(addresses)
}
func peerHintsFile(hash Hash) string {
	return filepath.Join(settings.Path, "flow-peer-hints", hash.HexString()+".json")
}

func (t *Torrent) restorePeerHints(handle *lt.Torrent) {
	network := peerHintsNetwork()
	peers := flow.ReadPeerHints(peerHintsFile(t.InfoHash), network, time.Now())
	if len(peers) == 0 {
		return
	}
	// One bounded injection per new handle, staggered across simultaneous starts.
	// Native min_reconnect_time and failure counters own subsequent attempts.
	timer := time.NewTimer(time.Duration(250+rand.IntN(1751)) * time.Millisecond)
	defer timer.Stop()
	select {
	case <-t.closeCh:
		return
	case <-timer.C:
	}
	if settings.CurrentFlow().PeerResumeHints && peerHintsNetwork() == network {
		_ = handle.RestorePeerHints(peers)
	}
}

func (t *Torrent) savePeerHints(handle *lt.Torrent, private bool) {
	if !settings.CurrentFlow().PeerResumeHints || settings.ReadOnly {
		return
	}
	if private {
		_ = os.Remove(peerHintsFile(t.InfoHash))
		return
	}
	if time.Since(t.peerHintsSavedAt) < time.Minute {
		return
	}
	peers, err := handle.ResumePeerHints()
	if err != nil || len(peers) == 0 {
		return
	}
	network := peerHintsNetwork()
	if network == "" {
		return
	}
	t.peerHintsSavedAt = time.Now()
	peerHintsDiskMu.Lock()
	defer peerHintsDiskMu.Unlock()
	dir := filepath.Dir(peerHintsFile(t.InfoHash))
	if err = os.MkdirAll(dir, 0700); err != nil {
		return
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	type saved struct {
		name string
		at   time.Time
	}
	var files []saved
	for _, entry := range entries {
		stem := strings.TrimSuffix(entry.Name(), ".json")
		if len(stem) != 40 || NewHashFromHex(stem).IsZero() || entry.IsDir() {
			continue
		}
		info, e := entry.Info()
		if e != nil || !info.Mode().IsRegular() {
			continue
		}
		name := filepath.Join(dir, entry.Name())
		if time.Since(info.ModTime()) > flow.PeerHintsTTL {
			_ = os.Remove(name)
			continue
		}
		files = append(files, saved{name, info.ModTime()})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].at.Before(files[j].at) })
	// One file per public torrent, at most 128 files / 1 MiB private state.
	name := peerHintsFile(t.InfoHash)
	exists := false
	for _, file := range files {
		if file.name == name {
			exists = true
		}
	}
	limit := 127
	if exists {
		limit = 128
	}
	for len(files) > limit {
		if os.Remove(files[0].name) != nil {
			return
		}
		files = files[1:]
	}
	_ = flow.WritePeerHints(name, flow.PeerHints{Version: 1, Network: network, SavedAt: time.Now(), Peers: peers})
}
