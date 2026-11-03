package torrstor

import (
	"bytes"
	"context"
	"errors"
	"io"
	"server/settings"
	"testing"
	"time"
)

func TestAuxiliaryReaderUsesVerifiedBytesWithoutPlayback(t *testing.T) {
	old := settings.BTsets()
	settings.StoreBTsets(settings.NewDefaultConfig())
	t.Cleanup(func() { settings.StoreBTsets(old) })
	c := mkCache(t, 3*pieceLen)
	defer c.close()
	data := bytes.Repeat([]byte{7}, int(pieceLen))
	if _, err := c.writePiece(0, 0, data); err != nil {
		t.Fatal(err)
	}
	c.SignalPieceComplete(0)
	reader, err := c.NewAuxiliaryReader(context.Background(), FileInfo{Offset: 0, Length: pieceLen})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.NewAuxiliaryReader(context.Background(), FileInfo{Offset: 0, Length: 1}); !errors.Is(err, ErrAuxiliaryDeferred) {
		t.Fatal("allowed concurrent sidecars for one torrent")
	}
	out, err := io.ReadAll(reader)
	if err != nil || !bytes.Equal(data, out) {
		t.Fatal("sidecar bytes", err)
	}
	if c.StreamingReaders() != 0 {
		t.Fatal("sidecar became playback")
	}
	reader.Close()
	if c.auxiliaryReserved() != 0 {
		t.Fatal("leaked reservation")
	}
}
func TestAuxiliaryCancellationAndPressureRelease(t *testing.T) {
	old := settings.BTsets()
	settings.StoreBTsets(settings.NewDefaultConfig())
	t.Cleanup(func() { settings.StoreBTsets(old) })
	c := mkCache(t, 2*pieceLen)
	defer c.close()
	ctx, cancel := context.WithCancel(context.Background())
	r, err := c.NewAuxiliaryReader(ctx, FileInfo{Offset: 0, Length: pieceLen})
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { _, err := r.Read(make([]byte, 1)); done <- err }()
	cancel()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("cancelled read succeeded")
		}
	case <-time.After(time.Second):
		t.Fatal("cancel did not unblock")
	}
	r.Close()
	c.backgroundLimited.Store(true)
	if _, err := c.NewAuxiliaryReader(context.Background(), FileInfo{Offset: 0, Length: 1}); !errors.Is(err, ErrAuxiliaryDeferred) {
		t.Fatal("admitted sidecar during pressure")
	}
}

func TestAuxiliaryChargesBoundaryPiecesAndHonorsPreloadAndClose(t *testing.T) {
	old := settings.BTsets()
	cfg := settings.NewDefaultConfig()
	cfg.CacheSize = 2 * pieceLen
	settings.StoreBTsets(cfg)
	t.Cleanup(func() { settings.StoreBTsets(old) })
	c := mkCache(t, 4*pieceLen)
	defer c.close()
	// Two bytes straddling a boundary charge two complete pieces.
	r, err := c.NewAuxiliaryReader(context.Background(), FileInfo{Offset: pieceLen - 1, Length: 2})
	if err != nil {
		t.Fatal(err)
	}
	if c.auxiliaryReserved() != 2*pieceLen {
		t.Fatal("uncharged boundary")
	}
	r.Close()
	c.resourceBudget.Store(pieceLen)
	if _, err = c.NewAuxiliaryReader(context.Background(), FileInfo{Offset: pieceLen - 1, Length: 2}); !errors.Is(err, ErrAuxiliaryDeferred) {
		t.Fatal("ignored effective spare budget")
	}
	c.resourceBudget.Store(0)
	c.preloadMu.Lock()
	c.preloadProtect = [][2]int{{0, 1}}
	c.preloadMu.Unlock()
	if _, err = c.NewAuxiliaryReader(context.Background(), FileInfo{Offset: 0, Length: 1}); !errors.Is(err, ErrAuxiliaryDeferred) {
		t.Fatal("competed with required startup")
	}
	c.preloadMu.Lock()
	c.preloadProtect = nil
	c.preloadMu.Unlock()
	r, err = c.NewAuxiliaryReader(context.Background(), FileInfo{Offset: 0, Length: 1})
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { _, err := r.Read(make([]byte, 1)); done <- err }()
	c.CancelAuxiliary()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("closed read succeeded")
		}
	case <-time.After(time.Second):
		t.Fatal("close did not cancel read")
	}
	r.Close()
	if c.auxiliaryReserved() != 0 || len(c.auxiliaryDemand(false)) != 0 {
		t.Fatal("demand remained after cancellation")
	}
}

func TestAuxiliaryDoesNotGrowCapacityForUnfilledForegroundWindow(t *testing.T) {
	old := settings.BTsets()
	cfg := settings.NewDefaultConfig()
	cfg.CacheSize = 32 << 20
	settings.StoreBTsets(cfg)
	t.Cleanup(func() { settings.StoreBTsets(old) })
	c := mkCache(t, 64<<20)
	defer c.close()
	foreground := NewReader(c, nil, FileInfo{Index: 0, Offset: 0, Length: 32 << 20})
	defer foreground.Close()
	for _, r := range [][2]int{foreground.headReserve(), foreground.tailReserve()} {
		for piece := r[0]; piece <= r[1]; piece++ {
			if _, err := c.writePiece(piece, 0, make([]byte, int(pieceLen))); err != nil {
				t.Fatal(err)
			}
			c.SignalPieceComplete(piece)
		}
	}
	before := c.capacity()
	r, err := c.NewAuxiliaryReader(context.Background(), FileInfo{Index: 1, Offset: 63 << 20, Length: 1})
	if err != nil {
		t.Fatal("unused physical cache was treated as occupied", err, c.streamingReserve())
	}
	defer r.Close()
	if c.capacity() != before {
		t.Fatal("auxiliary work enlarged foreground capacity", before, c.capacity())
	}
	if len(c.auxiliaryProtection()) != 1 {
		t.Fatal("admitted sidecar lost eviction protection")
	}
	c.backgroundLimited.Store(true)
	c.auxiliaryDemand(false)
	if r.ctx.Err() == nil {
		t.Fatal("pressure did not cancel sidecar")
	}
}

func TestAuxiliaryYieldsToMissingRequiredPins(t *testing.T) {
	old := settings.BTsets()
	settings.StoreBTsets(settings.NewDefaultConfig())
	t.Cleanup(func() { settings.StoreBTsets(old) })
	c := mkCache(t, 100*pieceLen)
	defer c.close()
	foreground := NewReader(c, nil, FileInfo{Offset: 0, Length: 50 * pieceLen})
	defer foreground.Close()
	if _, err := c.NewAuxiliaryReader(context.Background(), FileInfo{Offset: 99 * pieceLen, Length: 1}); !errors.Is(err, ErrAuxiliaryDeferred) {
		t.Fatal("competed with unverified required header/index", err)
	}
}

func TestAuxiliaryCompletionPinAndForegroundEvictionBudget(t *testing.T) {
	old := settings.BTsets()
	cfg := settings.NewDefaultConfig()
	cfg.CacheSize = 2 * pieceLen
	settings.StoreBTsets(cfg)
	t.Cleanup(func() { settings.StoreBTsets(old) })
	c := mkCache(t, 6*pieceLen)
	defer c.close()
	r, err := c.NewAuxiliaryReader(context.Background(), FileInfo{Offset: 5 * pieceLen, Length: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	// Hold the existing eviction owner while native completion races are
	// simulated, then exercise the exact callback and foreground trim paths.
	c.evicting.Store(true)
	defer c.evicting.Store(false)
	for _, index := range []int{0, 1, 5} {
		if _, err := c.writePiece(index, 0, bytes.Repeat([]byte{7}, int(pieceLen))); err != nil {
			t.Fatal(err)
		}
		c.SignalPieceComplete(index)
	}
	if c.evictComplete(5) {
		t.Fatal("native completion discarded an admitted sidecar pin")
	}
	c.evictPass()
	if r.ctx.Err() == nil || c.Filled() > cfg.CacheSize {
		t.Fatal("auxiliary work enlarged eviction capacity", c.Filled(), cfg.CacheSize)
	}
}

func TestAuxiliaryAdmissionCannotOutliveConcurrentCacheClose(t *testing.T) {
	old := settings.BTsets()
	settings.StoreBTsets(settings.NewDefaultConfig())
	t.Cleanup(func() { settings.StoreBTsets(old) })
	for attempt := 0; attempt < 32; attempt++ {
		c := mkCache(t, pieceLen)
		closed := make(chan struct{})
		go func() { c.close(); close(closed) }()
		r, err := c.NewAuxiliaryReader(context.Background(), FileInfo{Length: 1})
		<-closed
		if err == nil {
			if r.ctx.Err() == nil {
				t.Fatal("closed cache retained an auxiliary reader")
			}
			_ = r.Close()
		}
		if _, err := c.NewAuxiliaryReader(context.Background(), FileInfo{Length: 1}); !errors.Is(err, ErrAuxiliaryDeferred) {
			t.Fatal("closed cache admitted new work", err)
		}
		if c.auxiliaryReserved() != 0 {
			t.Fatal("closed cache retained reservation")
		}
	}
}
