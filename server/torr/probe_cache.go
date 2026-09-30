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

func (t *Torrent) restoreProbe(index int) bool {
	key, ok := t.probeKey(index)
	if !ok {
		return false
	}
	r, ok := mediaProbes.Get(key, time.Now())
	if !ok {
		return false
	}
	t.mu.Lock()
	t.BitRate, t.DurationSeconds, t.ProbeFileID = r.BitRate, r.Duration, index
	t.mu.Unlock()
	return true
}
