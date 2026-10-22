package torrstor

import (
	"bytes"
	"testing"
)

func TestContiguousBuffersShareDemandAnchorAndVerificationGate(t *testing.T) {
	c := mkCache(t, 2*pieceLen)
	for i := 0; i < 2; i++ {
		if _, err := c.writePiece(i, 0, bytes.Repeat([]byte{7}, int(pieceLen))); err != nil {
			t.Fatal(err)
		}
	}
	c.SignalPieceComplete(1)
	readable, verified := c.ContiguousBuffers(pieceLen/2, 2*pieceLen)
	if readable != pieceLen+pieceLen/2 || verified != 0 {
		t.Fatal("verification hole crossed", readable, verified)
	}
	c.SignalPieceComplete(0)
	readable, verified = c.ContiguousBuffers(pieceLen/2, 2*pieceLen)
	if readable != pieceLen+pieceLen/2 || verified != readable {
		t.Fatal("anchor mismatch", readable, verified)
	}
	c.InvalidatePiece(1)
	readable, verified = c.ContiguousBuffers(pieceLen/2, 2*pieceLen)
	if readable != pieceLen/2 || verified != readable {
		t.Fatal("invalidation was not sampled", readable, verified)
	}
	c.verifiedReads.Store(true)
	c.InvalidatePiece(0)
	if readable, verified = c.ContiguousBuffers(0, 2*pieceLen); readable != 0 || verified != 0 {
		t.Fatal("verification policy bypassed")
	}
}
