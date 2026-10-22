package flow

import (
	"encoding/json"
	"testing"
)

func TestStartupHistoryCloneIncludesSupplyPointers(t *testing.T) {
	piece, unchoked := 0, 1
	s := StartupHistory{HeadBlockerPiece: &piece, HeadSupply: &StartupPieceSupply{Suppliers: 2, UnchokedSuppliers: &unchoked}}
	copy := s.Clone()
	*copy.HeadBlockerPiece = 8
	*copy.HeadSupply.UnchokedSuppliers = 9
	copy.HeadSupply.Suppliers = 10
	if piece != 0 || unchoked != 1 || s.HeadSupply.Suppliers != 2 {
		t.Fatal("snapshot mutation changed retained state")
	}
}

func TestStartupSupplyRecordFitsRetentionLimit(t *testing.T) {
	piece, n := 7899, 512
	supply := &StartupPieceSupply{SampleAgeMs: 4999, Suppliers: 512, UnchokedSuppliers: &n, Truncated: true, QueueKnown: true, Priority: 7, RequestedBlocks: 1024, UnrequestedBlocks: 1024, FinishedBlocks: 1024, WritingBlocks: 1024, ReceivingBytes: 1 << 30, OldestRequestAgeMs: 300000}
	e := HistoryEvent{Type: "startup_sample", Startup: &StartupHistory{HeadSupply: supply, TailSupply: supply, HeadBlockerPiece: &piece, TailBlockerPiece: &piece, BlockedOn: "HEAD_AND_INDEX", WaitExtended: true, WaitBudgetMs: 300000}}
	data, err := json.Marshal(e)
	if err != nil || len(data)+1 > HistoryRecordBytes || !validHistoryEvent(e) {
		t.Fatal("bounded startup record rejected")
	}
}
