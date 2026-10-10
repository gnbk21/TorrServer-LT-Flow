package flow

import (
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"math/bits"
	"os"
	"path/filepath"
	"strconv"
)

// V2File hashes are metadata, never filename identities. PieceLayer contains
// SHA-256 nodes at the piece layer and may be absent while peers acquire them.
type V2File struct {
	Offset     int64  `json:"offset"`
	Size       int64  `json:"size"`
	Root       string `json:"root"`
	PieceLayer string `json:"piece_layer"`
}

type merkleStack struct {
	nodes [64][32]byte
	count uint64
}

func pairHash(a, b [32]byte) [32]byte {
	var data [64]byte
	copy(data[:32], a[:])
	copy(data[32:], b[:])
	return sha256.Sum256(data[:])
}
func (s *merkleStack) add(h [32]byte) {
	level := 0
	for n := s.count; n&1 != 0; n >>= 1 {
		h = pairHash(s.nodes[level], h)
		level++
	}
	s.nodes[level] = h
	s.count++
}
func (s *merkleStack) paddedRoot(zero [32]byte) [32]byte {
	if s.count == 0 {
		return zero
	}
	for s.count&(s.count-1) != 0 {
		s.add(zero)
	}
	return s.nodes[bits.TrailingZeros64(s.count)]
}
func digest32(text string) ([32]byte, bool) {
	var h [32]byte
	if len(text) != 64 {
		return h, false
	}
	_, err := hex.Decode(h[:], []byte(text))
	return h, err == nil
}

// merkleBytes hashes only actual file bytes. BEP 52 missing leaves are zero
// hashes, not hashes of zero-filled blocks. Working memory is one 16 KiB block
// and a logarithmic stack, independent of the torrent or piece size.
func merkleBytes(r io.Reader, size, leaves int64) ([32]byte, error) {
	if size <= 0 || leaves <= 0 || leaves&(leaves-1) != 0 || size > (leaves<<14) {
		return [32]byte{}, errors.New("invalid merkle geometry")
	}
	var stack merkleStack
	var block [16 << 10]byte
	for remaining := size; remaining > 0; {
		n := min(remaining, int64(len(block)))
		if _, err := io.ReadFull(r, block[:n]); err != nil {
			return [32]byte{}, err
		}
		stack.add(sha256.Sum256(block[:n]))
		remaining -= n
	}
	for int64(stack.count) < leaves {
		stack.add([32]byte{})
	}
	return stack.paddedRoot([32]byte{}), nil
}

// VerifyMetadataPieces verifies a resume bitmap against BOTH algorithms for
// hybrids. A piece layer must first match the file root. Missing/corrupt layers,
// short files, symlinks, changed bytes or incompatible geometry advertise no
// corresponding data. Padding is virtual and never stored as a second cache.
func VerifyMetadataPieces(dir string, pieceLength, totalSize int64, v1 [][20]byte, files []V2File) []byte {
	if len(files) == 0 {
		return VerifyPieceFiles(dir, pieceLength, totalSize, v1)
	}
	if pieceLength < 16<<10 || pieceLength > 1<<30 || pieceLength&(pieceLength-1) != 0 || totalSize <= 0 {
		return nil
	}
	numPieces := (totalSize-1)/pieceLength + 1
	if numPieces > 4_000_000 || (len(v1) != 0 && int64(len(v1)) != numPieces) {
		return nil
	}
	bitmap := make([]byte, (numPieces+7)/8)
	buffer := make([]byte, 64<<10)
	lastEnd := int64(0)
	for _, meta := range files {
		if meta.Offset < lastEnd || meta.Offset%pieceLength != 0 || meta.Size <= 0 || meta.Offset > totalSize-meta.Size {
			return nil
		}
		lastEnd = meta.Offset + meta.Size
		root, ok := digest32(meta.Root)
		if !ok {
			return nil
		}
		n := (meta.Size-1)/pieceLength + 1
		var layer []byte
		if n > 1 {
			if int64(len(meta.PieceLayer)) != n*64 {
				continue
			} // acquired later, still unknown
			var err error
			layer, err = hex.DecodeString(meta.PieceLayer)
			if err != nil {
				continue
			}
			zero := [32]byte{}
			for blocks := pieceLength >> 14; blocks > 1; blocks >>= 1 {
				zero = pairHash(zero, zero)
			}
			var tree merkleStack
			for i := int64(0); i < n; i++ {
				var node [32]byte
				copy(node[:], layer[i*32:(i+1)*32])
				tree.add(node)
			}
			if tree.paddedRoot(zero) != root {
				continue
			}
		}
		for i := int64(0); i < n; i++ {
			piece := meta.Offset/pieceLength + i
			size := min(pieceLength, meta.Size-i*pieceLength)
			expected := root
			leaves := pieceLength >> 14
			if n == 1 {
				leaves = 1 << bits.Len64(uint64((size-1)>>14))
			} else {
				copy(expected[:], layer[i*32:(i+1)*32])
			}
			path := filepath.Join(dir, strconv.FormatInt(piece, 10))
			st, err := os.Lstat(path)
			if err != nil || !st.Mode().IsRegular() || st.Size() != size {
				continue
			}
			file, err := os.Open(path)
			if err != nil {
				continue
			}
			opened, statErr := file.Stat()
			if statErr != nil || !opened.Mode().IsRegular() || !os.SameFile(st, opened) || opened.Size() != size {
				_ = file.Close()
				continue
			}
			digest, readErr := merkleBytes(file, size, leaves)
			valid := readErr == nil && digest == expected
			if valid && len(v1) > 0 {
				_, err = file.Seek(0, io.SeekStart)
				hash := sha1.New()
				var read int64
				if err == nil {
					read, err = io.CopyBuffer(hash, io.LimitReader(file, size+1), buffer)
				}
				// SHA-1 includes the hybrid's alignment padding; SHA-256 does not.
				for padding := min(pieceLength, totalSize-piece*pieceLength) - size; padding > 0; {
					chunk := min(padding, int64(len(buffer)))
					clear(buffer[:chunk])
					_, _ = hash.Write(buffer[:chunk])
					padding -= chunk
				}
				valid = err == nil && read == size && [20]byte(hash.Sum(nil)) == v1[piece]
			}
			closeErr := file.Close()
			if valid && closeErr == nil {
				bitmap[piece/8] |= 1 << uint(piece%8)
			}
		}
	}
	return bitmap
}
