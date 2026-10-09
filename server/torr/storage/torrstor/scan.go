package torrstor

import (
	"path/filepath"
	"server/flow"
)

// ScanHavePieces verifies existing disk pieces against exact metadata lengths
// and hashes. Unknown metadata never produces a trusted resume bitmap.
func ScanHavePieces(infoHash [20]byte, numPieces int, pieceLength, totalSize int64, hashes [][20]byte) []byte {
	if !useDisk() || numPieces != len(hashes) {
		return nil
	}
	dir := filepath.Join(savePath(), hashHex(infoHash))
	return flow.VerifyPieceFiles(dir, pieceLength, totalSize, hashes)
}
