package flow

import (
	"bytes"
	"context"
	"encoding/hex"
	"math/bits"
)

// ReuseFile contains authenticated content identity only. Names, tracker URLs,
// swarm wire IDs and release titles must never establish cross-torrent identity.
// Owners additionally enforce a process-wide descriptor/layer budget.
type ReuseFile struct {
	Root                      [32]byte
	Offset, Size, PieceLength int64
	Pieces                    [][32]byte
}

func NewReuseFile(meta V2File, pieceLength, totalSize int64) *ReuseFile {
	if pieceLength < 16<<10 || pieceLength > 8<<20 || pieceLength&(pieceLength-1) != 0 || meta.Offset < 0 || meta.Offset%pieceLength != 0 || meta.Size <= 0 || meta.Size > totalSize || meta.Offset > totalSize-meta.Size {
		return nil
	}
	root, ok := digest32(meta.Root)
	if !ok || root == ([32]byte{}) {
		return nil
	}
	n := (meta.Size-1)/pieceLength + 1
	if n > 65536 {
		return nil
	}
	f := &ReuseFile{Root: root, Offset: meta.Offset, Size: meta.Size, PieceLength: pieceLength}
	if n == 1 {
		f.Pieces = [][32]byte{root}
		return f
	}
	if int64(len(meta.PieceLayer)) != n*64 {
		return nil
	}
	layer, err := hex.DecodeString(meta.PieceLayer)
	if err != nil {
		return nil
	}
	f.Pieces = make([][32]byte, n)
	var tree merkleStack
	for i := range f.Pieces {
		copy(f.Pieces[i][:], layer[i*32:(i+1)*32])
		tree.add(f.Pieces[i])
	}
	zero := [32]byte{}
	for blocks := pieceLength >> 14; blocks > 1; blocks >>= 1 {
		zero = pairHash(zero, zero)
	}
	if tree.paddedRoot(zero) != root {
		return nil
	}
	return f
}

func (f *ReuseFile) Compatible(other *ReuseFile) bool {
	return f != nil && other != nil && f.Root == other.Root && f.Size == other.Size && f.PieceLength == other.PieceLength
}

func (f *ReuseFile) PieceBytes(index int) int64 {
	if f == nil || index < 0 || index >= len(f.Pieces) {
		return 0
	}
	return min(f.PieceLength, f.Size-int64(index)*f.PieceLength)
}

// Bytes copied from a verified cache are rehashed immediately before native
// submission. Native then independently verifies SHA-256 and hybrid SHA-1.
func (f *ReuseFile) Verify(ctx context.Context, index int, data []byte) bool {
	if f.PieceBytes(index) != int64(len(data)) || len(data) == 0 || ctx.Err() != nil {
		return false
	}
	leaves := f.PieceLength >> 14
	if len(f.Pieces) == 1 {
		leaves = 1 << bits.Len64(uint64((len(data)-1)>>14))
	}
	digest, err := merkleBytes(bytes.NewReader(data), int64(len(data)), leaves)
	return err == nil && digest == f.Pieces[index] && ctx.Err() == nil
}
