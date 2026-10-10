package flow

type SparseEvidence struct {
	Metadata, Fresh, Truncated, HasWindow  bool
	Peers, Candidates, Suppliers, Unchoked int
	ReadableBytes                          int64
	DownloadRate, ConsumptionRate          float64
}

type WaitExplanation struct {
	Reason     string   `json:"reason"`
	Confidence string   `json:"confidence"`
	Actions    []string `json:"actions"`
}

// Confidence describes this local observation. It never proves availability
// across the swarm, a storage bottleneck, or decoder-visible rebuffering.
func ExplainSparse(e SparseEvidence) WaitExplanation {
	x := WaitExplanation{Reason: SparseReason(e), Confidence: "unknown", Actions: []string{"CAPTURE_DIAGNOSTICS"}}
	switch x.Reason {
	case "METADATA":
		x.Actions = []string{"WAIT_METADATA", "CHECK_DISCOVERY"}
	case "READY":
		x.Confidence, x.Actions = "high", []string{}
	case "THROUGHPUT":
		x.Confidence, x.Actions = "medium", []string{"BUILD_RESERVE", "CHECK_NETWORK"}
	case "DISCOVERY":
		x.Confidence, x.Actions = "high", []string{"CHECK_DISCOVERY", "CHECK_NETWORK"}
	case "CONNECTION":
		x.Confidence, x.Actions = "high", []string{"WAIT_CONNECTION", "CHECK_NETWORK"}
	case "MISSING_CONNECTED":
		x.Confidence, x.Actions = "high", []string{"WAIT_SUPPLIERS", "CHECK_DISCOVERY"}
	case "UNCHOKE":
		x.Confidence, x.Actions = "high", []string{"WAIT_UNCHOKE", "CAPTURE_DIAGNOSTICS"}
	case "FIRST_BLOCK":
		x.Confidence, x.Actions = "medium", []string{"WAIT_BLOCK", "CAPTURE_DIAGNOSTICS"}
	}
	return x
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
