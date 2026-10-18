package torr

import (
	"server/flow"
	storageState "server/torr/storage/state"
)

// Count the same whole-piece gate as the preload, including shared boundary
// pieces. Received bytes alone do not imply successful piece verification.
func startupProgress(snapshot map[int]storageState.ItemState, headFirst, headLast, tailFirst, tailLast int, pieceLength int64) flow.StartupHistory {
	s := flow.StartupHistory{BlockedOn: "NONE"}
	add := func(first, last int, tail bool) {
		for p := first; p <= last; p++ {
			if tail && p >= headFirst && p <= headLast {
				continue
			}
			item := snapshot[p]
			bytes := min(pieceLength, max(int64(0), item.Size))
			verified := int64(0)
			if item.Completed {
				bytes, verified = pieceLength, pieceLength
			}
			if tail {
				s.TailTargetBytes += pieceLength
				s.TailBytes += bytes
				s.TailVerifiedBytes += verified
				if !item.Completed {
					s.MissingTailPieces++
					if s.TailBlockerPiece == nil {
						piece := p
						s.TailBlockerPiece = &piece
					}
				}
			} else {
				s.HeadTargetBytes += pieceLength
				s.HeadBytes += bytes
				s.HeadVerifiedBytes += verified
				if !item.Completed {
					s.MissingHeadPieces++
					if s.HeadBlockerPiece == nil {
						piece := p
						s.HeadBlockerPiece = &piece
					}
				}
			}
		}
	}
	add(headFirst, headLast, false)
	add(tailFirst, tailLast, true)
	switch {
	case s.MissingHeadPieces > 0 && s.MissingTailPieces > 0:
		s.BlockedOn = "HEAD_AND_INDEX"
	case s.MissingHeadPieces > 0:
		s.BlockedOn = "HEAD"
	case s.MissingTailPieces > 0:
		s.BlockedOn = "INDEX"
	}
	return s
}
