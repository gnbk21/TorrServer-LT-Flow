package torr

import (
	"server/flow"
	"time"
)

var mediaProbes flow.ProbeCache

func (t *Torrent) probeKey(index int) (flow.ProbeKey, bool) {
	f := t.fileByID(index)
	if f == nil {
		return flow.ProbeKey{}, false
	}
	return flow.ProbeKey{Hash: t.Hash().HexString(), Index: index, Size: f.Length, Path: f.Path}, true
}

func (t *Torrent) restoreProbe(index int, activeOnly bool) bool {
	key, ok := t.probeKey(index)
	if !ok {
		return false
	}
	r, ok := mediaProbes.Get(key, time.Now())
	if !ok {
		return false
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if activeOnly && t.flowStartup.FileIndex != 0 && t.flowStartup.FileIndex != index {
		return true // metadata is cached, but a newer file owns the visible startup
	}
	t.BitRate, t.DurationSeconds, t.ProbeFileID = r.BitRate, r.Duration, index
	if t.flowStartup.FileIndex == index {
		t.flowStartup.ProbeSuccess, t.flowStartup.ProbeCached = true, true
	}
	return true
}
