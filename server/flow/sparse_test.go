package flow

import "testing"

func TestSparseReasonDoesNotOverstateAvailability(t *testing.T) {
	tests := []struct {
		input SparseEvidence
		want  string
	}{
		{SparseEvidence{}, "METADATA"},
		{SparseEvidence{Metadata: true}, "UNKNOWN"},
		{SparseEvidence{Metadata: true, Fresh: true}, "DISCOVERY"},
		{SparseEvidence{Metadata: true, Fresh: true, Candidates: 1}, "CONNECTION"},
		{SparseEvidence{Metadata: true, Fresh: true, Peers: 1}, "UNKNOWN"},
		{SparseEvidence{Metadata: true, Fresh: true, Peers: 1, HasWindow: true}, "MISSING_CONNECTED"},
		{SparseEvidence{Metadata: true, Fresh: true, Peers: 1, HasWindow: true, Truncated: true}, "UNKNOWN"},
		{SparseEvidence{Metadata: true, Fresh: true, Peers: 1, HasWindow: true, Suppliers: 1}, "UNCHOKE"},
		{SparseEvidence{Metadata: true, Fresh: true, Peers: 1, HasWindow: true, Suppliers: 1, Truncated: true}, "UNKNOWN"},
		{SparseEvidence{Metadata: true, Fresh: true, Peers: 1, HasWindow: true, Suppliers: 1, Unchoked: 1, Truncated: true}, "FIRST_BLOCK"},
		{SparseEvidence{Metadata: true, Fresh: true, Peers: 1, HasWindow: true, Suppliers: 1, Unchoked: 1}, "FIRST_BLOCK"},
		{SparseEvidence{Metadata: true, ReadableBytes: 1}, "READY"},
		{SparseEvidence{Metadata: true, ReadableBytes: 1, ConsumptionRate: 100, DownloadRate: 50}, "THROUGHPUT"},
	}
	for _, test := range tests {
		if got := SparseReason(test.input); got != test.want {
			t.Fatalf("%+v: %s, want %s", test.input, got, test.want)
		}
	}
}
