package flow

import "math"

// DemandDeadline estimates when bytes ahead will be consumed. It preserves an
// immediate blocked piece and leaves ordinary priority work queued. Native
// deadlines may hedge stalled blocks; this is not a completion-order promise.
// qualifiedRate must come from metadata or stable consumption observations.
func DemandDeadline(position int, pieceLength int64, qualifiedRate float64) (int, bool) {
	if position <= 0 {
		return 0, true
	}
	if pieceLength <= 0 || qualifiedRate <= 0 || math.IsNaN(qualifiedRate) || math.IsInf(qualifiedRate, 0) {
		return 0, false
	}
	// Account for the native scheduler's one-second polling horizon. Bound far
	// deadlines so a slow/large-piece torrent still has a finite forward pipeline.
	due := float64(position)*float64(pieceLength)/qualifiedRate*1000 - 1000
	return int(math.Max(50, math.Min(30000, due))), true
}
