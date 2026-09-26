package torr

import (
	"errors"
	"sync/atomic"

	"server/torr/storage/torrstor"
)

var flowPaused atomic.Bool

type FlowTrayStatus struct {
	ServerState   string  `json:"server_state"`
	ActiveTorrent string  `json:"active_torrent"`
	DownloadRate  float64 `json:"download_rate"`
	BufferSeconds float64 `json:"buffer_seconds"`
	PeerCount     int     `json:"peer_count"`
}

func FlowIsPaused() bool { return flowPaused.Load() }

func SetFlowPaused(paused bool) error {
	if bts == nil || bts.Session() == nil {
		return errors.New("torrent engine is not running")
	}
	flowPaused.Store(paused)
	var firstErr error
	for _, tor := range bts.ListTorrents() {
		if tor == nil || tor.LTHandle() == nil {
			continue
		}
		var err error
		if paused {
			err = tor.LTHandle().Pause()
		} else {
			err = tor.LTHandle().Resume()
		}
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func SnapshotFlowTray() FlowTrayStatus {
	snapshot := FlowTrayStatus{ServerState: "RUNNING"}
	if bts == nil || bts.Session() == nil {
		snapshot.ServerState = "STARTING"
		return snapshot
	}
	if flowPaused.Load() {
		snapshot.ServerState = "PAUSED"
	}
	for _, tor := range bts.ListTorrents() {
		if tor == nil {
			continue
		}
		cache := torrstor.Global().CacheByHash([20]byte(tor.Hash()))
		if cache == nil || cache.StreamingReaders() == 0 {
			continue
		}
		status := tor.Status()
		snapshot.ActiveTorrent = tor.Title
		if snapshot.ActiveTorrent == "" {
			snapshot.ActiveTorrent = tor.Name()
		}
		snapshot.DownloadRate = status.DownloadSpeed
		snapshot.PeerCount = status.ActivePeers
		for _, session := range tor.FlowStatus() {
			if session.ActiveReaders <= 0 {
				continue
			}
			snapshot.BufferSeconds = session.BufferAheadSeconds
			break
		}
		return snapshot
	}
	return snapshot
}
