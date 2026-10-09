package flow

import (
	"crypto/sha1"
	"io"
	"os"
	"path/filepath"
	"strconv"
)

// VerifyPieceFiles returns an LSB-first bitmap. Size is only a prerequisite:
// every advertised piece, including the short final piece, must match metadata.
// Memory stays bounded to one copy buffer rather than the torrent's contents.
func VerifyPieceFiles(dir string, pieceLength, totalSize int64, hashes [][20]byte) []byte {
	if pieceLength <= 0 || totalSize <= 0 || len(hashes) == 0 ||
		(totalSize-1)/pieceLength+1 != int64(len(hashes)) {
		return nil
	}
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		return nil
	}
	bitmap := make([]byte, (len(hashes)+7)/8)
	buffer := make([]byte, 64<<10)
	for i, expectedHash := range hashes {
		path := filepath.Join(dir, strconv.Itoa(i))
		st, err := os.Lstat(path)
		expectedSize := min(pieceLength, totalSize-int64(i)*pieceLength)
		if err != nil || !st.Mode().IsRegular() || st.Size() != expectedSize {
			continue
		}
		file, err := os.Open(path)
		if err != nil {
			continue
		}
		hash := sha1.New()
		n, readErr := io.CopyBuffer(hash, io.LimitReader(file, expectedSize+1), buffer)
		closeErr := file.Close()
		if readErr == nil && closeErr == nil && n == expectedSize &&
			[20]byte(hash.Sum(nil)) == expectedHash {
			bitmap[i/8] |= 1 << uint(i%8)
		}
	}
	return bitmap
}
