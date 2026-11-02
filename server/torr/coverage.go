package torr

import (
	"server/flow"
	"server/torr/storage/torrstor"
	"time"
)

// FlowCoverage joins immutable native evidence with bounded cache observations.
func (t *Torrent) FlowCoverage(sessions []FlowSessionStatus, startup FlowStartupStatus) *flow.Coverage {
	now := time.Now()
	out := &flow.Coverage{SchemaVersion: 1, SampledAtMS: now.UnixMilli(), Pieces: []flow.CoveragePiece{}, Frontiers: []flow.CoverageFrontier{}}
	sample := t.SparseStatus()
	out.AvailabilitySampledAtMS = sample.SampledAtMs
	out.Truncated = sample.Truncated || sample.UrgentTruncated
	cache := torrstor.Global().CacheByHash(t.StorageHash())
	if cache == nil || cache.PieceLength <= 0 {
		return out
	}
	out.Known, out.PieceLength = true, cache.PieceLength
	pieces := make(map[int]bool)
	var indices []int
	add := func(index int) {
		if index < 0 || index >= cache.NumPieces || pieces[index] {
			return
		}
		if len(indices) >= 256 {
			out.Truncated = true
			return
		}
		pieces[index] = true
		indices = append(indices, index)
	}
	for _, session := range sessions {
		if session.ActiveReaders == 0 {
			continue
		}
		file := t.fileByID(session.FileIndex)
		if file == nil {
			continue
		}
		first := int((file.Offset + session.DemandOffsetBytes) / cache.PieceLength)
		if len(out.Frontiers) >= 8 {
			out.Truncated = true
			break
		}
		out.Frontiers = append(out.Frontiers, flow.CoverageFrontier{FileIndex: session.FileIndex, Piece: first, ReadableBytes: session.ReadableContiguousBytes, VerifiedBytes: session.VerifiedContiguousBytes, SampledAtMS: session.BufferSampledAtMs})
		for index := first; index < first+24; index++ {
			add(index)
		}
	}
	requirements, limited := cache.CoverageRequirements()
	out.Truncated = out.Truncated || limited
	for _, required := range requirements {
		for index := required.First; index <= required.Last && index < required.First+8; index++ {
			add(index)
		}
		add(required.Last)
		if required.Last-required.First >= 8 {
			out.Truncated = true
		}
	}
	for _, window := range sample.Windows[:min(8, len(sample.Windows))] {
		for i := range window.Availability {
			add(window.FirstPiece + i)
		}
	}
	startupBlocking := startup.State != "READY" && startup.State != "PLAYING" && startup.State != "FAILED" && startup.State != "TIMEOUT" && startup.State != "CANCELLED"
	if startup.Progress != nil && startupBlocking {
		for _, p := range []*int{startup.Progress.HeadBlockerPiece, startup.Progress.TailBlockerPiece} {
			if p != nil {
				add(*p)
			}
		}
	}
	resident := cache.ResidentCoverage(indices)
	fresh := sample.Known && now.UnixMilli() >= sample.SampledAtMs && now.UnixMilli()-sample.SampledAtMs < 5000
	for _, index := range indices {
		state := resident[index]
		row := flow.CoveragePiece{Piece: index, Resident: state[0], Verified: state[1]}
		for _, required := range requirements {
			if index >= required.First && index <= required.Last {
				row.RequiredHead = row.RequiredHead || required.Head
				row.RequiredIndex = row.RequiredIndex || required.Index
			}
		}
		if startup.Progress != nil && startupBlocking {
			row.Blocker = startup.Progress.HeadBlockerPiece != nil && index == *startup.Progress.HeadBlockerPiece || startup.Progress.TailBlockerPiece != nil && index == *startup.Progress.TailBlockerPiece
		}
		for _, session := range sessions {
			if session.ActiveReaders > 0 && session.Blocker.Known && session.Blocker.Piece == index && session.Blocker.Reason != "READY" {
				row.Blocker = true
			}
		}
		if fresh {
			for _, window := range sample.Windows {
				offset := index - window.FirstPiece
				if offset >= 0 && offset < len(window.Availability) {
					n := window.Availability[offset]
					row.Suppliers = &n
					break
				}
			}
			for _, urgent := range sample.Urgent {
				if urgent.Piece == index {
					row.QueueKnown = true
					row.Requested = urgent.Requested
					row.ReceivingBytes = urgent.ReceivingBytes
					row.Writing = urgent.Writing
					row.Finished = urgent.Finished
					break
				}
			}
		}
		out.Pieces = append(out.Pieces, row)
	}
	return out
}
