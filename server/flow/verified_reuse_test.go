package flow

import (
	"bytes"
	"context"
	"encoding/hex"
	"testing"
)

func TestVerifiedReuseRequiresFullRootGeometryAndRehashedBytes(t *testing.T) {
	data := bytes.Repeat([]byte{0x72}, 33017)
	first, err := merkleBytes(bytes.NewReader(data[:32768]), 32768, 2)
	if err != nil {
		t.Fatal(err)
	}
	last, err := merkleBytes(bytes.NewReader(data[32768:]), 249, 2)
	if err != nil {
		t.Fatal(err)
	}
	root := pairHash(first, last)
	layer := hex.EncodeToString(first[:]) + hex.EncodeToString(last[:])
	meta := V2File{Offset: 32768, Size: int64(len(data)), Root: hex.EncodeToString(root[:]), PieceLayer: layer}
	f := NewReuseFile(meta, 32768, 32768+int64(len(data)))
	if f == nil || !f.Verify(context.Background(), 0, data[:32768]) || !f.Verify(context.Background(), 1, data[32768:]) {
		t.Fatal("authenticated short-tail geometry was lost")
	}
	changed := append([]byte(nil), data[:32768]...)
	changed[17] ^= 1
	if f.Verify(context.Background(), 0, changed) || f.Verify(context.Background(), 1, append(data[32768:], 0)) {
		t.Fatal("changed or padded file bytes were accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if f.Verify(ctx, 0, data[:32768]) {
		t.Fatal("cancelled reuse was admitted")
	}
	other := *f
	other.Offset = 0
	if !f.Compatible(&other) {
		t.Fatal("compatible content at another native offset was rejected")
	}
	other.Size++
	if f.Compatible(&other) {
		t.Fatal("same root with incompatible file size was accepted")
	}
	other = *f
	other.PieceLength *= 2
	if f.Compatible(&other) {
		t.Fatal("incompatible piece mapping was accepted")
	}
	for _, mutate := range []func(*V2File){
		func(m *V2File) { m.Offset++ }, func(m *V2File) { m.PieceLayer = "" }, func(m *V2File) { m.PieceLayer = hex.EncodeToString(make([]byte, 64)) }, func(m *V2File) { m.Root = m.Root[:40] },
	} {
		bad := meta
		mutate(&bad)
		if NewReuseFile(bad, 32768, 100000) != nil {
			t.Fatal("unproven metadata accepted")
		}
	}
}

func TestVerifiedReuseSmallFileUsesItsOwnMerkleHeight(t *testing.T) {
	data := bytes.Repeat([]byte{0x42}, 10001)
	root, err := merkleBytes(bytes.NewReader(data), int64(len(data)), 1)
	if err != nil {
		t.Fatal(err)
	}
	f := NewReuseFile(V2File{Size: int64(len(data)), Root: hex.EncodeToString(root[:])}, 32768, int64(len(data)))
	if f == nil || !f.Verify(context.Background(), 0, data) {
		t.Fatal("single short-file root used torrent piece padding")
	}
}
