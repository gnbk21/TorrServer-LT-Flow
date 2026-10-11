package flow

import (
	"bytes"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func referenceMerkle(data []byte, leaves int) [32]byte {
	nodes := make([][32]byte, leaves)
	for i := 0; i < len(data); i += 16 << 10 {
		nodes[i>>14] = sha256.Sum256(data[i:min(i+(16<<10), len(data))])
	}
	for len(nodes) > 1 {
		next := make([][32]byte, len(nodes)/2)
		for i := range next {
			var joined [64]byte
			copy(joined[:32], nodes[i*2][:])
			copy(joined[32:], nodes[i*2+1][:])
			next[i] = sha256.Sum256(joined[:])
		}
		nodes = next
	}
	return nodes[0]
}
func rootHex(h [32]byte) string { return hex.EncodeToString(h[:]) }

func TestMerkleLeavesAndShortFiles(t *testing.T) {
	for _, size := range []int{1, 10001, 16384, 17001, 32768, 49001} {
		data := bytes.Repeat([]byte{byte(size)}, size)
		leaves := 1
		for leaves*16384 < size {
			leaves *= 2
		}
		got, err := merkleBytes(bytes.NewReader(data), int64(size), int64(leaves))
		if err != nil || got != referenceMerkle(data, leaves) {
			t.Fatalf("size %d: root mismatch %v", size, err)
		}
		if _, err = merkleBytes(bytes.NewReader(data[:size-1]), int64(size), int64(leaves)); err == nil {
			t.Fatal("short read accepted")
		}
	}
}

func TestV2ResumeAuthenticatesPieceLayerAndHybridPadding(t *testing.T) {
	const piece = 32768
	dir := t.TempDir()
	first := bytes.Repeat([]byte{0x34}, piece)
	tail := bytes.Repeat([]byte{0x56}, 617)
	next := bytes.Repeat([]byte{0x78}, 10001)
	layer := append([]byte(nil), first...)
	layer = append(layer, tail...)
	root := referenceMerkle(layer, 4)
	firstNode := referenceMerkle(first, 2)
	tailNode := referenceMerkle(tail, 2)
	files := []V2File{{Offset: 0, Size: int64(len(layer)), Root: rootHex(root), PieceLayer: rootHex(firstNode) + rootHex(tailNode)},
		{Offset: piece * 2, Size: int64(len(next)), Root: rootHex(referenceMerkle(next, 1))}}
	data := [][]byte{first, tail, next}
	v1 := make([][20]byte, len(data))
	for i, payload := range data {
		if err := os.WriteFile(filepath.Join(dir, strconv.Itoa(i)), payload, 0600); err != nil {
			t.Fatal(err)
		}
		padded := make([]byte, piece)
		copy(padded, payload)
		v1[i] = sha1.Sum(padded)
	}
	check := func(want byte, hashes [][20]byte) {
		t.Helper()
		bitmap := VerifyMetadataPieces(dir, piece, piece*3, hashes, files)
		if len(bitmap) != 1 || bitmap[0] != want {
			t.Fatalf("want bitmap %03b, got %v", want, bitmap)
		}
	}
	check(7, nil)
	check(7, v1)
	v1[1][0] ^= 1
	check(5, v1)
	v1[1][0] ^= 1
	files[0].PieceLayer = rootHex(firstNode) + rootHex(firstNode)
	check(4, nil)
	files[0].PieceLayer = ""
	check(4, nil)
	files[0].PieceLayer = rootHex(firstNode) + rootHex(tailNode)
	if err := os.WriteFile(filepath.Join(dir, "1"), bytes.Repeat([]byte{0}, len(tail)), 0600); err != nil {
		t.Fatal(err)
	}
	check(5, nil)
	if err := os.WriteFile(filepath.Join(dir, "2"), next[:len(next)-1], 0600); err != nil {
		t.Fatal(err)
	}
	check(1, nil)
	files[1].Offset = 1
	if VerifyMetadataPieces(dir, piece, piece*3, nil, files) != nil {
		t.Fatal("overlapping geometry accepted")
	}
}

func TestV2ResumePadsPieceLayerAtCorrectHeight(t *testing.T) {
	const piece = 65536
	data := bytes.Repeat([]byte{0xaa}, piece*2+1)
	file := V2File{Offset: 0, Size: int64(len(data)), Root: rootHex(referenceMerkle(data, 16))}
	dir := t.TempDir()
	for i := 0; i < 3; i++ {
		payload := data[i*piece : min((i+1)*piece, len(data))]
		file.PieceLayer += rootHex(referenceMerkle(payload, 4))
		if err := os.WriteFile(filepath.Join(dir, strconv.Itoa(i)), payload, 0600); err != nil {
			t.Fatal(err)
		}
	}
	bitmap := VerifyMetadataPieces(dir, piece, piece*3, nil, []V2File{file})
	if len(bitmap) != 1 || bitmap[0] != 7 {
		t.Fatalf("wrong zero subtree height: %v", bitmap)
	}
}
