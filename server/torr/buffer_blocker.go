package torr

import (
	"server/lt"
	"time"
)

// BufferBlocker is a copied, bounded explanation for the actual demand piece.
// Finished blocks still require hash verification; request age may be unknown.
type BufferBlocker struct {
	Known              bool   `json:"known"`
	Piece              int    `json:"piece"`
	PieceLength        int64  `json:"piece_length"`
	Blocks             int    `json:"blocks"`
	Unrequested        int    `json:"unrequested"`
	Requested          int    `json:"requested"`
	Writing            int    `json:"writing"`
	Finished           int    `json:"finished"`
	OldestRequestAgeMs int64  `json:"oldest_request_age_ms"`
	Suppliers          *int   `json:"suppliers,omitempty"`
	Reason             string `json:"reason"`
}

func bufferBlocker(sample lt.SparseSnapshot, piece int, pieceLength int64, now time.Time) BufferBlocker {
	b := BufferBlocker{Reason: "UNKNOWN", OldestRequestAgeMs: -1}
	age := now.Sub(time.UnixMilli(sample.SampledAtMs))
	if piece < 0 || pieceLength <= 0 || !sample.Known || sample.Truncated || sample.UrgentTruncated ||
		sample.RequestAgeTruncated || age < 0 || age >= 5*time.Second {
		return b
	}
	for _, urgent := range sample.Urgent {
		if urgent.Piece != piece {
			continue
		}
		b.Known, b.Piece, b.PieceLength = true, piece, pieceLength
		b.Blocks, b.Unrequested, b.Requested = urgent.Blocks, urgent.Unrequested, urgent.Requested
		b.Writing, b.Finished, b.OldestRequestAgeMs = urgent.Writing, urgent.Finished, urgent.OldestRequestAgeMs
		b.Reason = "WAIT_BLOCKS"
		if urgent.Verified {
			b.Reason = "READY"
		} else if urgent.Finished+urgent.Writing == urgent.Blocks {
			b.Reason = "VERIFYING"
		}
		for _, window := range sample.Windows {
			i := piece - window.FirstPiece
			if i >= 0 && i < len(window.Availability) {
				count := window.Availability[i]
				b.Suppliers = &count
				break
			}
		}
		return b
	}
	return b
}
