package torr

import (
	storageState "server/torr/storage/state"
	"testing"
)

func TestStartupProgressSeparatesBoundaryVerificationFromReceivedBytes(t *testing.T) {
	snap := map[int]storageState.ItemState{0: {Completed: true}, 1: {Size: 100}, 3: {Size: 100}}
	s := startupProgress(snap, 0, 1, 3, 3, 100)
	if s.HeadBytes != 200 || s.HeadVerifiedBytes != 100 || s.TailBytes != 100 || s.TailVerifiedBytes != 0 || s.BlockedOn != "HEAD_AND_INDEX" || *s.HeadBlockerPiece != 1 || *s.TailBlockerPiece != 3 {
		t.Fatalf("incorrect partial/hash-wait evidence: %+v", s)
	}
	snap[1] = storageState.ItemState{Completed: true}
	s = startupProgress(snap, 0, 1, 3, 3, 100)
	if s.BlockedOn != "INDEX" || s.MissingHeadPieces != 0 || s.MissingTailPieces != 1 {
		t.Fatalf("index straggler not isolated: %+v", s)
	}
	snap[3] = storageState.ItemState{Completed: true}
	s = startupProgress(snap, 0, 1, 1, 3, 100)
	if s.TailTargetBytes != 200 || s.MissingTailPieces != 1 || *s.TailBlockerPiece != 2 {
		t.Fatalf("overlap double-counted: %+v", s)
	}
}

func TestStartupProgressIncludesPieceZeroAsBlockerAndDeduplicatesShortFile(t *testing.T) {
	s := startupProgress(nil, 0, 0, 0, 0, 100)
	if s.HeadBlockerPiece == nil || *s.HeadBlockerPiece != 0 || s.TailTargetBytes != 0 || s.BlockedOn != "HEAD" {
		t.Fatalf("piece-zero/short-file evidence: %+v", s)
	}
}

func TestBootstrapEvidenceDoesNotBlameUnscheduledHeadRemainder(t *testing.T) {
	snap := map[int]storageState.ItemState{0: {Completed: true}, 1: {Completed: true}}
	s := startupGateProgress(snap, 0, 3, 1, 8, 8, 100, true)
	if !s.Bootstrap || s.BlockedOn != "INDEX" || s.HeadTargetBytes != 200 || s.MissingHeadPieces != 0 {
		t.Fatalf("unscheduled phase 2 falsely blamed: %+v", s)
	}
	s = startupGateProgress(snap, 0, 3, 1, 2, 3, 100, true)
	if s.BlockedOn != "NONE" || s.TailTargetBytes != 0 {
		t.Fatalf("overlap added pieces not in bootstrap gate: %+v", s)
	}
	s = startupGateProgress(snap, 0, 3, 1, 8, 8, 100, false)
	if s.Bootstrap || s.BlockedOn != "HEAD_AND_INDEX" || s.HeadTargetBytes != 400 {
		t.Fatalf("phase 2 evidence incorrect: %+v", s)
	}
}
