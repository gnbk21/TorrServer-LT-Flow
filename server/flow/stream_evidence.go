package flow

// StreamService describes completed block requests, including remote queueing.
// It is not network RTT. Percentiles are histogram bucket upper bounds.
type StreamService struct {
	Transport     string `json:"transport"`
	AddressFamily string `json:"address_family"`
	Samples       uint64 `json:"samples"`
	P50Ms         int64  `json:"p50_ms"`
	P95Ms         int64  `json:"p95_ms"`
}

// StreamEvidence contains no media, client, or peer identifiers. Native times
// use its monotonic torrent lifetime, independent of the Go preload lifetime.
type StreamEvidence struct {
	RequestPolicy         *RequestPolicy  `json:"request_policy,omitempty"`
	RecoveryScan          *RecoveryScan   `json:"recovery_scan,omitempty"`
	Known                 bool            `json:"known"`
	FirstDialMs           int64           `json:"first_dial_ms"`
	FirstConnectMs        int64           `json:"first_connect_ms"`
	FirstHandshakeMs      int64           `json:"first_handshake_ms"`
	FirstPayloadMs        int64           `json:"first_payload_ms"`
	ConnectDurationMs     int64           `json:"connect_duration_ms"`
	HandshakeDurationMs   int64           `json:"handshake_duration_ms"`
	Service               []StreamService `json:"service"`
	UsefulBytes           uint64          `json:"useful_bytes"`
	RedundantBytes        uint64          `json:"redundant_bytes"`
	ObsoleteBytes         uint64          `json:"obsolete_bytes"`
	RecoveryRequests      uint64          `json:"recovery_requests"`
	RecoverySuppressed    uint64          `json:"recovery_suppressed"`
	ActiveHedges          int             `json:"active_hedges"`
	PeakHedges            int             `json:"peak_hedges"`
	ActiveHedgeBytes      int64           `json:"active_hedge_bytes"`
	BudgetChanges         uint64          `json:"budget_changes"`
	Frontiers             int             `json:"frontiers"`
	RecoveryScanTruncated bool            `json:"recovery_scan_truncated"`
}

// RequestPolicy describes bounded aggregate queue occupancy without peer IDs.
type RequestPolicy struct {
	Enabled              bool  `json:"enabled"`
	SampledPeers         int   `json:"sampled_peers"`
	QualifiedPeers       int   `json:"qualified_peers"`
	CapacityLimitedPeers int   `json:"capacity_limited_peers"`
	DesiredRequests      int64 `json:"desired_requests"`
	QueuedRequests       int64 `json:"queued_requests"`
	CapacityRequests     int64 `json:"capacity_requests"`
	HorizonMinMs         int64 `json:"horizon_min_ms"`
	HorizonMaxMs         int64 `json:"horizon_max_ms"`
	Truncated            bool  `json:"truncated"`
}

// RecoveryScan counts independent request decisions in the most recent bounded
// scan. These are snapshots, not lifetime failure counters.
type RecoveryScan struct {
	AgeMs             int64 `json:"age_ms"`
	Eligible          int   `json:"eligible"`
	UncertainRequest  int   `json:"uncertain_request"`
	StaleService      int   `json:"stale_service"`
	AlreadyDuplicated int   `json:"already_duplicated"`
	NotLate           int   `json:"not_late"`
	Receiving         int   `json:"receiving"`
}

func (s *StreamEvidence) Valid() bool {
	if s == nil {
		return true
	}
	if len(s.Service) > 6 || s.ActiveHedges < 0 || s.ActiveHedges > 16 ||
		s.PeakHedges < 0 || s.PeakHedges > 16 || s.ActiveHedgeBytes < 0 ||
		s.ActiveHedgeBytes > 256<<10 || s.Frontiers < 0 || s.Frontiers > 8 {
		return false
	}
	for _, value := range []int64{s.FirstDialMs, s.FirstConnectMs, s.FirstHandshakeMs, s.FirstPayloadMs, s.ConnectDurationMs, s.HandshakeDurationMs} {
		if value < -1 {
			return false
		}
	}
	if s.ObsoleteBytes > s.RedundantBytes || s.ActiveHedges > s.PeakHedges {
		return false
	}
	if p := s.RequestPolicy; p != nil {
		if p.SampledPeers < 0 || p.SampledPeers > 512 || p.QualifiedPeers < 0 ||
			p.QualifiedPeers > p.SampledPeers || p.CapacityLimitedPeers < 0 || p.CapacityLimitedPeers > p.SampledPeers ||
			p.DesiredRequests < 0 || p.DesiredRequests > p.CapacityRequests || p.QueuedRequests < 0 ||
			p.QueuedRequests > 512*65535*2 || p.CapacityRequests < 0 || p.CapacityRequests > 512*65535 ||
			p.HorizonMinMs < 0 || p.HorizonMaxMs < p.HorizonMinMs || p.HorizonMaxMs > 6000 ||
			(p.QualifiedPeers > 0 && p.HorizonMinMs < 250) ||
			(p.QualifiedPeers == 0 && (p.HorizonMinMs != 0 || p.HorizonMaxMs != 0)) {
			return false
		}
	}
	if r := s.RecoveryScan; r != nil {
		total := int64(0)
		for _, n := range []int{r.Eligible, r.UncertainRequest, r.StaleService, r.AlreadyDuplicated, r.NotLate, r.Receiving} {
			if n < 0 || n > 8192 {
				return false
			}
			total += int64(n)
		}
		if r.AgeMs < -1 || total > 8192 {
			return false
		}
	}
	seen := map[string]bool{}
	for _, row := range s.Service {
		if SafeTransport(row.Transport) != row.Transport || (row.AddressFamily != "IPV4" && row.AddressFamily != "IPV6") ||
			row.P50Ms < 0 || row.P95Ms < row.P50Ms || row.P95Ms > 300000 {
			return false
		}
		key := row.Transport + "/" + row.AddressFamily
		if seen[key] {
			return false
		}
		seen[key] = true
	}
	return true
}

type StartupPhase struct {
	Phase      string `json:"phase"`
	Basis      string `json:"basis"`
	DurationMs *int64 `json:"duration_ms,omitempty"`
}

// StartupPhases intentionally reports overlapping milestones, not an additive
// latency breakdown. Missing native evidence and unfinished operations stay
// unknown; cached probes are explicitly measured as zero.
func StartupPhases(native *StreamEvidence, metadata, bootstrap, probeStart, probeEnd, reserve, firstByte int64, cached bool) []StartupPhase {
	phases := make([]StartupPhase, 0, 9)
	add := func(phase, basis string, value int64) {
		row := StartupPhase{Phase: phase, Basis: basis}
		if value >= 0 {
			v := value
			row.DurationMs = &v
		}
		phases = append(phases, row)
	}
	value := func(v int64) int64 {
		if native == nil || !native.Known {
			return -1
		}
		return v
	}
	if native == nil {
		native = &StreamEvidence{FirstDialMs: -1, ConnectDurationMs: -1, HandshakeDurationMs: -1, FirstPayloadMs: -1}
	}
	add("DISCOVERY", "NATIVE_TORRENT", value(native.FirstDialMs))
	add("CONNECT", "OPERATION", value(native.ConnectDurationMs))
	add("HANDSHAKE", "OPERATION", value(native.HandshakeDurationMs))
	add("METADATA", "TORRENT", metadata)
	add("FIRST_PAYLOAD", "NATIVE_TORRENT", value(native.FirstPayloadMs))
	add("BOOTSTRAP", "PRELOAD", bootstrap)
	probe := int64(-1)
	if cached {
		probe = 0
	} else if probeStart >= 0 && probeEnd >= probeStart {
		probe = probeEnd - probeStart
	}
	add("PROBE", "OPERATION", probe)
	add("RESERVE", "PRELOAD", reserve)
	add("FIRST_BYTE", "PRELOAD", firstByte)
	return phases
}
