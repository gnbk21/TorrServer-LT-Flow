package torrstor

import (
	"server/cachefs"
	"server/flow"
)

// ScanHavePieces verifies existing disk pieces against exact metadata lengths
// and hashes. Unknown metadata never produces a trusted resume bitmap.
func ScanHavePieces(infoHash flow.InfoHash, numPieces int, pieceLength, totalSize int64, hashes [][20]byte) []byte {
	if !useDisk() || numPieces != len(hashes) {
		return nil
	}
	root, err := cachefs.Open(savePath(), false)
	if err != nil {
		return nil
	}
	defer root.Close()
	return flow.VerifyPieceFilesIn(root, hashHex(infoHash), pieceLength, totalSize, hashes)
}

func ScanMetadataPieces(hash flow.InfoHash, pieceLength, totalSize int64, hashes [][20]byte, files []flow.V2File) []byte {
	if !useDisk() {
		return nil
	}
	root, err := cachefs.Open(savePath(), false)
	if err != nil {
		return nil
	}
	defer root.Close()
	return flow.VerifyMetadataPiecesIn(root, hash.HexString(), pieceLength, totalSize, hashes, files)
}
