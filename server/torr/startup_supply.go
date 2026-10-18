package torr

import (
	"server/flow"
	"server/lt"
	"time"
)

func startupSparseRanges(startup FlowStartupStatus, numPieces int) [][2]int {
	if startup.Progress == nil || (startup.State != "BOOTSTRAP_PRELOAD" && startup.State != "ADAPTIVE_PREBUFFER") {
		return nil
	}
	var ranges [][2]int
	for _, piece := range []*int{startup.Progress.TailBlockerPiece, startup.Progress.HeadBlockerPiece} {
		if piece != nil && *piece >= 0 && *piece < numPieces && (len(ranges) == 0 || ranges[0][0] != *piece) {
			ranges = append(ranges, [2]int{*piece, 1})
		}
	}
	return ranges
}

func startupPieceSupply(sample lt.SparseSnapshot, piece *int, now time.Time) *flow.StartupPieceSupply {
	age := now.Sub(time.UnixMilli(sample.SampledAtMs))
	if piece == nil || !sample.Known || age < 0 || age >= 5*time.Second {
		return nil
	}
	var result *flow.StartupPieceSupply
	for _, window := range sample.Windows {
		index := *piece - window.FirstPiece
		if index < 0 || index >= len(window.Availability) {
			continue
		}
		result = &flow.StartupPieceSupply{SampleAgeMs: age.Milliseconds(), Suppliers: window.Availability[index], Truncated: sample.Truncated, OldestRequestAgeMs: -1}
		// Native unchoke count belongs to the entire sampled window. Only a
		// one-piece window makes it a supplier count for this exact blocker.
		if len(window.Availability) == 1 {
			unchoked := window.UnchokedSuppliers
			result.UnchokedSuppliers = &unchoked
		}
		break
	}
	if result == nil {
		return nil
	}
	for _, urgent := range sample.Urgent {
		if urgent.Piece != *piece {
			continue
		}
		result.QueueKnown = true
		result.Priority, result.RequestedBlocks, result.UnrequestedBlocks = urgent.Priority, urgent.Requested, urgent.Unrequested
		result.FinishedBlocks, result.WritingBlocks, result.ReceivingBytes = urgent.Finished, urgent.Writing, urgent.ReceivingBytes
		result.OldestRequestAgeMs = urgent.OldestRequestAgeMs
		result.Truncated = result.Truncated || sample.UrgentTruncated || sample.RequestAgeTruncated
		break
	}
	return result
}
