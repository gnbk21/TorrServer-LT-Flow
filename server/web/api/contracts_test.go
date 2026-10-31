package api

import (
	"encoding/json"
	"os"
	"server/diagnostics"
	"server/flow"
	"server/lt"
	"server/rutor/models"
	"server/searchstatus"
	sets "server/settings"
	"server/torr"
	"server/torr/state"
	"server/torr/storage/torrstor"
	"server/version"
	"testing"
	"time"
)

// The native test job emits real API DTOs and then runs frontend validators on
// this file. This tests a language boundary, rather than two copies of a fixture.
func TestExportFrontendContracts(t *testing.T) {
	output := os.Getenv("FLOW_CONTRACT_OUTPUT")
	if output == "" {
		t.Skip("contract export is requested by the native integration job")
	}
	hash := "0123456789012345678901234567890123456789"
	measuredDuration := int64(2)
	stream := &flow.StreamEvidence{Known: true, FirstDialMs: 0, FirstConnectMs: 5,
		FirstHandshakeMs: 10, FirstPayloadMs: 20, ConnectDurationMs: 5, HandshakeDurationMs: 5,
		Service:     []flow.StreamService{{Transport: "tcp", AddressFamily: "IPV4", Samples: 8, P50Ms: 100, P95Ms: 250}},
		UsefulBytes: 16384, RecoveryRequests: 1, PeakHedges: 1}
	stream.RequestPolicy = &flow.RequestPolicy{Enabled: true, SampledPeers: 2, QualifiedPeers: 1,
		DesiredRequests: 16, QueuedRequests: 12, CapacityRequests: 64, HorizonMinMs: 4000, HorizonMaxMs: 4000}
	stream.RecoveryScan = &flow.RecoveryScan{AgeMs: 100, Eligible: 1, UncertainRequest: 1}
	supply := flow.DeliveryEvidence{Mode: "DEMAND", Confidence: "high", Samples: 20,
		AgeMs: 100, LongRate: 2e6, DeficitBytes: 1e6, DeficitSamples: 20}
	reserve := flow.PlanStartupReserve(32<<20, 128<<20, 16<<20, 3e6, 45, supply)
	cases := []struct {
		Kind    string `json:"kind"`
		Payload any    `json:"payload"`
	}{
		{"settings-metadata", sets.SettingsMetadata()},
		{"settings", sets.BTSets{CacheSize: 64 << 20, ReaderReadAHead: 95, PreloadCache: 50, Flow: sets.DefaultFlowSettings()}},
		{"settings", sets.BTSets{CacheSize: 64 << 20, Flow: nil}},
		{"torrent", state.TorrentStatus{Hash: hash, Title: "Generated fixture", Stat: state.TorrentInDB}},
		{"library", LibraryResponse{Items: []state.TorrentStatus{{Hash: hash, Title: "Fixture", Stat: state.TorrentInDB}}, Total: 1, LibraryTotal: 1, Page: 1, Limit: 50, Categories: []string{}, SampledAt: "2026-10-10T00:00:00Z"}},
		{"active", map[string]any{"items": []activeTorrent{{Torrent: state.TorrentStatus{Hash: hash, Title: "Fixture", Stat: state.TorrentWorking}, Status: compactFlowStatus{Hash: hash, Sessions: []torr.FlowSessionStatus{}}}}, "sampled_at": "2026-10-10T00:00:00Z"}},
		{"flow", FlowStatusResponse{Hash: hash, Sparse: lt.SparseSnapshot{Known: true, Stream: stream},
			Startup: torr.FlowStartupStatus{Reserve: &reserve, Phases: flow.StartupPhases(stream, 0, 20, 20, 25, 50, 60, false)},
			Sessions: []torr.FlowSessionStatus{{Group: "contract", FileIndex: 1, State: "PLAYING",
				Supply: flow.AssessBufferSupply(3e6, 30e6, supply),
				Blocker: torr.BufferBlocker{Known: true, Piece: 0, PieceLength: 16 << 20, Blocks: 1024,
					Finished: 1022, Requested: 2, OldestRequestAgeMs: -1, Reason: "WAIT_BLOCKS"}}}}},
		{"flow", FlowStatusResponse{Hash: hash, Sessions: nil}},
		{"flow", FlowStatusResponse{Hash: hash, NextEpisodeWarmup: &torr.NextEpisodeStatus{Enabled: true, CurrentFileIndex: 1, NextEpisodeStatus: torrstor.NextEpisodeStatus{FileIndex: 2, State: "waiting", Reason: "PLAYBACK_RESERVE", BudgetBytes: 32 << 20}}}},
		{"flow", FlowStatusResponse{Hash: hash, SampledAt: "2026-10-10T00:00:00Z", Timeline: &flow.TimelineSnapshot{Source: "server", Events: []flow.TimelineEvent{{Time: time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC), Type: "seek", ElapsedMs: 1000, OperationMs: &measuredDuration, File: 1}, {Time: time.Date(2026, 10, 10, 0, 0, 1, 0, time.UTC), Type: "metadata", ElapsedMs: 1001, File: 1}}, Dropped: 0}, Sessions: []torr.FlowSessionStatus{{SessionID: "ABCDEFGHIJKLMNOPQRSTUV", FileIndex: 1, State: "PLAYING"}}}},
		{"flow", FlowStatusResponse{Hash: hash, StorageIO: lt.StorageIO{QueuedBytes: 16384, PeakQueueBytes: 32768, QueuedJobs: 2, CompletedJobs: 7, WaitP95Us: 128, CallbackP95Us: 256}, Sessions: []torr.FlowSessionStatus{{Group: "evidence", FileIndex: 1, State: "PLAYING", ReadableContiguousBytes: 32768, VerifiedContiguousBytes: 16384, Delivery: flow.DeliveryEvidence{DeficitBytes: 4096, DeficitSamples: 4}}}}},
		{"runtime", RuntimeStatus{Build: version.Identity(), BT: &torr.ClientStatusSnapshot{}, Memory: diagnostics.Memory()}},
		{"flow", FlowStatusResponse{Hash: hash, Integrity: &flow.IntegritySnapshot{Failures: 2, RecentFailures: 1, WindowSeconds: 60, LastFailureAtMs: 1792600000000}, Trackers: []torr.FlowTrackerDiagnostic{{ID: "fixture", Host: "fixture.example", Protocol: "https", Status: "ERROR", ErrorCategory: "HTTP", ErrorCode: 503, Operation: 16, FailureCount: 2, LastAt: time.Date(2026, 10, 22, 0, 0, 0, 0, time.UTC)}}, Sessions: []torr.FlowSessionStatus{{FileIndex: 1, State: "PLAYING", BufferSampledAtMs: 1792600000000, BufferOffsetBytes: 16384, ReadableContiguousBytes: 32768, VerifiedContiguousBytes: 16384}}}},
		{"storage", torr.StoragePreferencesSnapshot()},
		{"sessions", flow.SessionSummaries{Enabled: true, Items: []flow.SessionSummary{{Started: time.Date(2026, 10, 24, 0, 0, 0, 0, time.UTC), LastObserved: time.Date(2026, 10, 24, 0, 1, 0, 0, time.UTC), Samples: 2, BytesServed: 1024, MinimumBufferBytes: 256, MaximumBufferBytes: 512, PeakRecentReadStalls: 1, PeakRecentWaitP95Ms: 4, RangeRequests: 2, RangeCancels: 1, Incomplete: true}}}},
		{"search", detailedSearchResponse{Items: []*models.TorrentDetails{{Title: "generated", Hash: hash}}, Sources: []searchstatus.Status{{Source: "torznab", Index: 0, State: "ok"}}}},
		{"network", torr.FlowNetworkStatus{State: "NO_ADDRESS", Connectivity: "INTERNET_WAIT"}},
		{"preparation", torr.PreparationStatus{Jobs: []torr.PreparationJob{{ID: hash + ":1", Hash: hash, FileIndex: 1, State: "downloading", Length: 100, VerifiedBytes: 60, ContiguousBytes: 40}}, QuotaBytes: 4096 << 20, ReservedBytes: 16384}},
		{"sources", torr.WebSeedStatus{Sources: []torr.WebSeedSummary{{ID: "01234567890123456789012345678901", Origin: "https://fixture.example", Disabled: false, AllowLocal: false}}}},
		{"flow", FlowStatusResponse{Hash: hash, Sparse: lt.SparseSnapshot{Known: true, SampledAtMs: 1, SampledPeers: 1, UsefulPeers: 1, Urgent: []lt.UrgentPiece{{Piece: 4, Priority: 7, Blocks: 256, Unrequested: 100, Requested: 120, Writing: 6, Finished: 30, ReceivingBlocks: 1, ReceivingBytes: 8192}}, Windows: []lt.SparseWindow{{FirstPiece: 4, Availability: []int{1, 0}, UnchokedSuppliers: 1}}}}},
	}
	data, err := json.MarshalIndent(cases, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(output, data, 0600); err != nil {
		t.Fatal(err)
	}
}
