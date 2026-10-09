package torrstor

import (
	"bytes"
	"context"
	"io"
	"testing"
)

func TestBufferedReaderTracksConsumptionRatherThanPrefetch(t *testing.T) {
	c := mkCache(t, 2*pieceLen)
	for i := 0; i < 2; i++ {
		if _, err := c.writePiece(i, 0, bytes.Repeat([]byte{byte(i + 1)}, int(pieceLen))); err != nil {
			t.Fatal(err)
		}
		c.SignalPieceComplete(i)
	}
	r := NewReader(c, nil, FileInfo{Length: 2 * pieceLen}, "buffered-device")
	defer r.Close()
	r.TrackBufferedConsumption()
	if _, err := r.Read(make([]byte, pieceLen)); err != nil {
		t.Fatal(err)
	}
	if r.Offset() != 0 || r.currentPiece() != 0 {
		t.Fatal("prefetch moved the playhead", r.Offset(), r.currentPiece())
	}
	r.ConsumeBuffered(17)
	if r.Offset() != 17 || r.offset.Load() != pieceLen {
		t.Fatal("consumption and source position were conflated")
	}
	// The HTTP wrapper compensates for unread buffered bytes before calling Seek.
	if pos, err := r.Seek(17-pieceLen, io.SeekCurrent); err != nil || pos != 17 {
		t.Fatal(pos, err)
	}
	if r.Offset() != 17 {
		t.Fatal("seek did not reset logical consumption")
	}
	if _, err := r.Seek(2*pieceLen, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Read(make([]byte, 1)); err != io.EOF {
		t.Fatal("EOF semantics changed", err)
	}
}

func TestBufferedReaderReturnsAvailableBytesWithoutWaitingForFullBuffer(t *testing.T) {
	c := mkCache(t, 2*pieceLen)
	if _, err := c.writePiece(0, 0, bytes.Repeat([]byte{1}, int(pieceLen))); err != nil {
		t.Fatal(err)
	}
	c.SignalPieceComplete(0)
	r := NewReader(c, nil, FileInfo{Length: 2 * pieceLen})
	defer r.Close()
	r.TrackBufferedConsumption()
	ctx, cancel := context.WithCancel(context.Background())
	r.SetContext(ctx)
	n, err := r.Read(make([]byte, 2*pieceLen))
	if err != nil || int64(n) != pieceLen {
		t.Fatal("read waited past an available prefix", n, err)
	}
	cancel()
	if _, err := r.Read(make([]byte, 1)); err == nil {
		t.Fatal("cancelled wait did not stop")
	}
}
