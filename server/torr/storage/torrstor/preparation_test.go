package torrstor

import (
	"bytes"
	"os"
	"path/filepath"
	"server/settings"
	"testing"
)

func TestPreparationSharesCacheRetainsAndMigrates(t *testing.T) {
	prev := settings.BTsets()
	settings.StoreBTsets(&settings.BTSets{CacheSize: 16 << 10})
	defer settings.StoreBTsets(prev)
	s := NewStorage()
	var hash [20]byte
	hash[0] = 9
	s.callbackOpen(1, hash, 3, 16<<10)
	c := s.CacheByHash(hash)
	c.totalSize.Store(3 * 16 << 10)
	data := bytes.Repeat([]byte{7}, 16<<10)
	if _, err := c.writePiece(1, 0, data); err != nil {
		t.Fatal(err)
	}
	c.SignalPieceComplete(1)
	root := t.TempDir()
	if err := s.ConfigurePreparation(hash, PreparationStorage{Root: root, Ranges: map[string]PreparationRange{"job": {1, 2, true}}}); err != nil {
		t.Fatal(err)
	}
	if c.pieces[1].mem != nil || c.pieces[1].disk == nil {
		t.Fatal("retained a second RAM store")
	}
	if c.Filled() != 0 || c.evictComplete(1) || c.prunePartial(1) {
		t.Fatal("retained disk bytes were evicted or charged to RAM cache")
	}
	if verified, prefix := c.PreparationProgress(16<<10, 2*16<<10); verified != 16<<10 || prefix != verified {
		t.Fatalf("%d %d", verified, prefix)
	}
	if err := c.FlushPreparation(1, 1); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(root, hashHex(hash), "1"))
	if err != nil || !bytes.Equal(data, got) {
		t.Fatalf("migration %v", err)
	}
	if want := c.preparationDemand(); len(want) != 1 || want[0] != 2 {
		t.Fatalf("demand %v", want)
	}
	s.callbackClose(1)
	if _, err := os.Stat(filepath.Join(root, hashHex(hash), "1")); err != nil {
		t.Fatal("close removed retained pieces", err)
	}
}

func TestPreparationCleanupFencesReaderRegistration(t *testing.T) {
	c := newCache(NewStorage(), 1, [20]byte{}, 1, 16<<10)
	r := NewReader(c, nil, FileInfo{Length: 16 << 10})
	if c.BeginPreparationCleanup() {
		t.Fatal("cleanup accepted an active player")
	}
	r.Close()
	if !c.BeginPreparationCleanup() {
		t.Fatal("cleanup did not begin")
	}
	if NewReader(c, nil, FileInfo{Length: 16 << 10}) != nil {
		t.Fatal("new reader raced cleanup")
	}
}
