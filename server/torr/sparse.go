package torr

import (
	"server/flow"
	"server/lt"
	"server/settings"
	"server/torr/storage/torrstor"
	"sort"
	"time"
)

func (t *Torrent) sampleSparse(handle *lt.Torrent, metadata bool, pieceLength int64, numPieces int) {
	if !settings.CurrentFlow().MetricsEnabled {
		return
	}
	t.flowMu.Lock()
	type demand struct {
		start int
		at    time.Time
	}
	var demands []demand
	if metadata && pieceLength > 0 {
		for _, s := range t.flowSessions {
			if s.ActiveReaders > 0 && s.Group != torrstor.ProbeReaderGroup {
				first := int((s.fileOffset + s.PlaybackOffsetBytes) / pieceLength)
				if first >= 0 && first < numPieces {
					demands = append(demands, demand{first, s.lastSeen})
				}
			}
		}
	}
	t.flowMu.Unlock()
	sort.Slice(demands, func(i, j int) bool {
		if demands[i].at.Equal(demands[j].at) {
			return demands[i].start < demands[j].start
		}
		return demands[i].at.After(demands[j].at)
	})
	var ranges [][2]int
	seen := make(map[int]bool)
	for _, d := range demands {
		if seen[d.start] {
			continue
		}
		seen[d.start] = true
		ranges = append(ranges, [2]int{d.start, min(32, numPieces-d.start)})
		if len(ranges) == 8 {
			break
		}
	}
	if snapshot, err := handle.SampleSparse(ranges); err == nil {
		t.flowMu.Lock()
		t.sparse = snapshot
		t.flowMu.Unlock()
	}
}

func (t *Torrent) SparseStatus() lt.SparseSnapshot {
	if t == nil {
		return lt.SparseSnapshot{}
	}
	t.flowMu.Lock()
	defer t.flowMu.Unlock()
	return t.sparse // immutable after publication
}

func sparseSessionEvidence(sample lt.SparseSnapshot, first int, readable int64, download, consumption float64, peers, candidates int) flow.SparseEvidence {
	age := time.Since(time.UnixMilli(sample.SampledAtMs))
	e := flow.SparseEvidence{Metadata: true, Fresh: sample.Known && age >= 0 && age < 5*time.Second,
		Truncated: sample.Truncated, Peers: peers, Candidates: candidates, ReadableBytes: readable, DownloadRate: download, ConsumptionRate: consumption}
	for _, window := range sample.Windows {
		if window.FirstPiece == first && len(window.Availability) > 0 {
			e.HasWindow, e.Suppliers, e.Unchoked = true, window.Availability[0], window.UnchokedSuppliers
			break
		}
	}
	return e
}
