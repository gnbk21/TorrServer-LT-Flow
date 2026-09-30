package torrstor

import (
	"bytes"
	"context"
	"testing"
	"time"

	"server/flow"
	"server/settings"
)

func TestParkedReadReconcilesLateCompletionAndCancels(t *testing.T) {
	old := settings.BTsets()
	settings.StoreBTsets(&settings.BTSets{CacheSize: 64 * flow.MiB})
	t.Cleanup(func() { settings.StoreBTsets(old) })
	s := NewStorage()
	h := mkHash(0xE3)
	s.callbackOpen(3, h, 4, pieceBlockSize)
	c := s.CacheByHash(h)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	count := 0
	if !c.waitForBytes(ctx, 0, 0, func() { count++; _, _ = s.callbackWrite(3, 0, 0, bytes.Repeat([]byte{7}, pieceBlockSize)) }) || count != 1 {
		t.Fatal("parked read did not reconcile missing data")
	}
	ctx, cancel = context.WithCancel(context.Background())
	cancel()
	if c.waitForBytes(ctx, 1, 0, func() { t.Fatal("reconciled cancelled request") }) {
		t.Fatal("cancelled reader continued waiting")
	}
}

func TestContiguousAvailableStopsAtPartialPieceHole(t *testing.T) {
	old := settings.BTsets()
	settings.StoreBTsets(&settings.BTSets{CacheSize: 64 * flow.MiB})
	t.Cleanup(func() { settings.StoreBTsets(old) })
	s := NewStorage()
	h := mkHash(0xE1)
	s.callbackOpen(1, h, 4, 4*pieceBlockSize)
	c := s.CacheByHash(h)
	if c == nil {
		t.Fatal("cache missing")
	}
	block := bytes.Repeat([]byte{1}, pieceBlockSize)
	if _, err := s.callbackWrite(1, 0, 0, block); err != nil {
		t.Fatal(err)
	}
	if got := c.ContiguousAvailable(0, 3*pieceBlockSize); got != pieceBlockSize {
		t.Fatalf("first block: got %d", got)
	}
	if _, err := s.callbackWrite(1, 0, 2*pieceBlockSize, block); err != nil {
		t.Fatal(err)
	}
	if got := c.ContiguousAvailable(0, 3*pieceBlockSize); got != pieceBlockSize {
		t.Fatalf("hole should stop run: got %d", got)
	}
	if _, err := s.callbackWrite(1, 0, pieceBlockSize, block); err != nil {
		t.Fatal(err)
	}
	if got := c.ContiguousAvailable(0, 3*pieceBlockSize); got != 3*pieceBlockSize {
		t.Fatalf("contiguous blocks: got %d", got)
	}
}

func TestAdaptiveWindowUsesExistingCacheBudget(t *testing.T) {
	old := settings.BTsets()
	settings.StoreBTsets(&settings.BTSets{CacheSize: 512 * flow.MiB, ReaderReadAHead: 95, Flow: settings.DefaultFlowSettings()})
	t.Cleanup(func() { settings.StoreBTsets(old) })
	s := NewStorage()
	h := mkHash(0xE2)
	s.callbackOpen(2, h, 256, 4*flow.MiB)
	c := s.CacheByHash(h)
	if c == nil {
		t.Fatal("cache missing")
	}
	_, maxAhead := c.baseReaderWindowPieces()
	c.SetFlowMediaEstimate("phone", 0, flow.Estimate{BytesPerSecond: 2 * float64(flow.MiB)})
	c.SetFlowDownloadRate(10 * float64(flow.MiB))
	_, healthy := c.readerWindowPieces()
	if healthy <= 0 || healthy >= maxAhead {
		t.Fatalf("adaptive ahead=%d, budget maximum=%d", healthy, maxAhead)
	}
	c.SetFlowDownloadRate(float64(flow.MiB) / 2)
	_, weak := c.readerWindowPieces()
	if weak <= healthy || weak > maxAhead {
		t.Fatalf("weak swarm ahead=%d, healthy=%d, max=%d", weak, healthy, maxAhead)
	}
}
