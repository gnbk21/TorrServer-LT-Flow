package flow

type SparseEvidence struct {
	Metadata, Fresh, Truncated, HasWindow  bool
	Peers, Candidates, Suppliers, Unchoked int
	ReadableBytes                          int64
	DownloadRate, ConsumptionRate          float64
}

// SparseReason describes local evidence, never global torrent availability.
func SparseReason(e SparseEvidence) string {
	if !e.Metadata {
		return "METADATA"
	}
	if e.ReadableBytes > 0 {
		if e.ConsumptionRate > 0 && e.DownloadRate < e.ConsumptionRate {
			return "THROUGHPUT"
		}
		return "READY"
	}
	if !e.Fresh {
		return "UNKNOWN"
	}
	if e.Peers == 0 {
		if e.Candidates > 0 {
			return "CONNECTION"
		}
		return "DISCOVERY"
	}
	if !e.HasWindow {
		// A shifted playhead or the bounded window limit can leave this
		// piece unsampled. That is not evidence of local cache/probe work.
		return "UNKNOWN"
	}
	if e.Suppliers == 0 {
		if e.Truncated {
			return "UNKNOWN"
		}
		return "MISSING_CONNECTED"
	}
	if e.Unchoked == 0 {
		if e.Truncated {
			return "UNKNOWN" // an omitted supplier may be unchoked
		}
		return "UNCHOKE"
	}
	return "FIRST_BLOCK"
}
