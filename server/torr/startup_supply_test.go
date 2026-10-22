package torr

import (
	"reflect"
	"server/flow"
	"server/lt"
	"testing"
	"time"
)

func TestStartupSparseDemandTracksOnlyActiveBlockers(t *testing.T) {
	head, tail := 0, 7899
	s := FlowStartupStatus{State: "BOOTSTRAP_PRELOAD", Progress: &flow.StartupHistory{HeadBlockerPiece: &head, TailBlockerPiece: &tail}}
	if got := startupSparseRanges(s, 8000); !reflect.DeepEqual(got, [][2]int{{7899, 1}, {0, 1}}) {
		t.Fatalf("startup demand: %v", got)
	}
	s.State = "TIMEOUT"
	if got := startupSparseRanges(s, 8000); len(got) != 0 {
		t.Fatal("expired demand retained")
	}
	s.State = "ADAPTIVE_PREBUFFER"
	s.Progress.TailBlockerPiece = &head
	if got := startupSparseRanges(s, 8000); !reflect.DeepEqual(got, [][2]int{{0, 1}}) {
		t.Fatal("shared blocker duplicated")
	}
	if got := startupSparseRanges(s, 0); len(got) != 0 {
		t.Fatal("out-of-range demand")
	}
}

func TestStartupSupplyDistinguishesMissingPeersAndUnrequestedBlocks(t *testing.T) {
	now := time.Unix(100, 0)
	piece := 4
	sample := lt.SparseSnapshot{Known: true, SampledAtMs: now.Add(-time.Second).UnixMilli(), Windows: []lt.SparseWindow{{FirstPiece: 4, Availability: []int{0}, UnchokedSuppliers: 0}}, Urgent: []lt.UrgentPiece{{Piece: 4, Priority: 7, Unrequested: 1024, OldestRequestAgeMs: -1}}}
	e := startupPieceSupply(sample, &piece, now)
	if e == nil || e.Suppliers != 0 || e.UnchokedSuppliers == nil || *e.UnchokedSuppliers != 0 || !e.QueueKnown || e.UnrequestedBlocks != 1024 || e.SampleAgeMs != 1000 {
		t.Fatalf("missing supply evidence: %+v", e)
	}
	sample.SampledAtMs = now.Add(-6 * time.Second).UnixMilli()
	if startupPieceSupply(sample, &piece, now) != nil {
		t.Fatal("stale observation used")
	}
	sample.SampledAtMs = now.Add(time.Second).UnixMilli()
	if startupPieceSupply(sample, &piece, now) != nil {
		t.Fatal("future observation used")
	}
	sample.SampledAtMs = now.UnixMilli()
	sample.Windows[0] = lt.SparseWindow{FirstPiece: 3, Availability: []int{7, 2}, UnchokedSuppliers: 6}
	sample.Urgent = nil
	e = startupPieceSupply(sample, &piece, now)
	if e == nil || e.Suppliers != 2 || e.UnchokedSuppliers != nil || e.QueueKnown {
		t.Fatal("window-level unchoke count misrepresented as piece supply")
	}
	sample.Truncated = true
	if !startupPieceSupply(sample, &piece, now).Truncated {
		t.Fatal("partial peer census hidden")
	}
}
