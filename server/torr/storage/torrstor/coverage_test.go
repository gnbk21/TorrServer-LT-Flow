package torrstor

import (
	"bytes"
	"testing"
)

func TestResidentCoverageDoesNotPromoteUnverifiedBlocks(t *testing.T) {
	c := mkCache(t, 300*pieceLen)
	if _, err := c.writePiece(0, 0, bytes.Repeat([]byte{1}, int(pieceLen))); err != nil {
		t.Fatal(err)
	}
	first := c.ResidentCoverage([]int{0, 1})
	if first[0] != [2]bool{true, false} || first[1] != [2]bool{} {
		t.Fatalf("unverified snapshot: %v", first)
	}
	c.SignalPieceComplete(0)
	if c.ResidentCoverage([]int{0})[0] != [2]bool{true, true} {
		t.Fatal("verified data omitted")
	}
	indices := make([]int, 300)
	for i := range indices {
		indices[i] = i
	}
	if got := c.ResidentCoverage(indices); len(got) > 256 {
		t.Fatal("unbounded coverage")
	}
	if c.StreamingReaders() != 0 {
		t.Fatal("observation created a reader")
	}
}

func TestCoverageRequirementsUsesActualRefinedIndexPinsWithoutDemand(t *testing.T) {
	c := mkCache(t, 300*pieceLen)
	defer c.close()
	c.SetPreloadReserve([][2]int{{0, 2}, {299, 299}, {150, 151}})
	rows, truncated := c.CoverageRequirements()
	if truncated || len(rows) != 3 || !rows[0].Head || !rows[1].Index || !rows[2].Index || rows[2].First != 150 {
		t.Fatal(rows, truncated)
	}
	rows[0].First = 99
	current, _ := c.CoverageRequirements()
	if current[0].First != 0 || c.Filled() != 0 || c.StreamingReaders() != 0 {
		t.Fatal("observation changed cache or demand")
	}
	readers := []*Reader{}
	for index := 0; index < 8; index++ {
		readers = append(readers, NewReader(c, nil, FileInfo{Index: index, Offset: int64(index*30) * pieceLen, Length: 20 * pieceLen}))
	}
	defer func() {
		for _, r := range readers {
			_ = r.Close()
		}
	}()
	rows, truncated = c.CoverageRequirements()
	if !truncated || len(rows) > 8 {
		t.Fatal("unbounded required ranges", len(rows), truncated)
	}
}
