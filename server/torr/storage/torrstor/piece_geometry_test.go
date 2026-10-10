package torrstor

import (
	"bytes"
	"server/lt"
	"testing"
)

func TestV2ShortFileTailNeedsActualBytesAndNativeVerification(t *testing.T) {
	s := NewStorage()
	s.callbackOpen(1, mkHash(61), 3, 32768)
	s.callbackSize(1, 3*32768)
	s.callbackGeometry(1, []lt.PieceSize{{Piece: 0, Size: 17001}, {Piece: 1, Size: 1}})
	c := s.lookup(1)
	if _, err := c.writePiece(0, 0, bytes.Repeat([]byte{7}, 16384)); err != nil {
		t.Fatal(err)
	}
	c.SignalPieceComplete(0)
	if c.Have(0) {
		t.Fatal("missing short block was treated as verified")
	}
	if _, err := c.writePiece(0, 16384, bytes.Repeat([]byte{8}, 617)); err != nil {
		t.Fatal(err)
	}
	if c.Have(0) {
		t.Fatal("writes alone established v2 verification")
	}
	c.SignalPieceComplete(0)
	if !c.Have(0) || c.VerifiedContiguousAvailable(0, 17001) != 17001 {
		t.Fatal("short v2 file tail never completed")
	}
	if _, err := c.writePiece(1, 0, []byte{9}); err != nil {
		t.Fatal(err)
	}
	c.SignalPieceComplete(1)
	if !c.Have(1) {
		t.Fatal("single byte v2 file never completed")
	}
	// Invalid native geometry cannot replace the already published layout.
	s.callbackGeometry(1, []lt.PieceSize{{Piece: 0, Size: 32769}})
	if c.pieces[0].expectedSize() != 17001 {
		t.Fatal("invalid geometry was accepted")
	}
	c.InvalidatePiece(0)
	if c.Have(0) {
		t.Fatal("hash failure retained trusted v2 bytes")
	}
}
