package torr

import (
	"server/lt"
	"testing"
	"time"
)

func TestBufferBlockerUsesActualDemandAndFreshCompleteSample(t *testing.T) {
	now := time.Unix(1000, 0)
	for _, length := range []int64{256 << 10, 4 << 20, 16 << 20} {
		blocks := int(length / (16 << 10))
		s := lt.SparseSnapshot{Known: true, SampledAtMs: now.UnixMilli(),
			Windows: []lt.SparseWindow{{FirstPiece: 12, Availability: []int{3, 0}}},
			Urgent:  []lt.UrgentPiece{{Piece: 12, Blocks: blocks, Finished: blocks - 2, Requested: 2, OldestRequestAgeMs: 4000}}}
		b := bufferBlocker(s, 12, length, now)
		if !b.Known || b.PieceLength != length || b.Requested != 2 || b.Suppliers == nil || *b.Suppliers != 3 || b.Reason != "WAIT_BLOCKS" {
			t.Fatal(b)
		}
		if bufferBlocker(s, 13, length, now).Known {
			t.Fatal("another piece was labelled as the current blocker")
		}
		s.Urgent[0].Finished, s.Urgent[0].Requested = blocks, 0
		if b = bufferBlocker(s, 12, length, now); b.Reason != "VERIFYING" {
			t.Fatal("received data reported as verified", b)
		}
		s.Urgent[0].Verified = true
		if b = bufferBlocker(s, 12, length, now); b.Reason != "READY" {
			t.Fatal(b)
		}
		for _, mutate := range []func(*lt.SparseSnapshot){
			func(s *lt.SparseSnapshot) { s.Known = false },
			func(s *lt.SparseSnapshot) { s.Truncated = true },
			func(s *lt.SparseSnapshot) { s.UrgentTruncated = true },
			func(s *lt.SparseSnapshot) { s.RequestAgeTruncated = true },
			func(s *lt.SparseSnapshot) { s.SampledAtMs = now.Add(-5 * time.Second).UnixMilli() },
			func(s *lt.SparseSnapshot) { s.SampledAtMs = now.Add(time.Second).UnixMilli() },
		} {
			copy := s
			mutate(&copy)
			if bufferBlocker(copy, 12, length, now).Known {
				t.Fatal("stale, partial or unknown evidence accepted")
			}
		}
	}
}
