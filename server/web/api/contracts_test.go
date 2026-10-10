package api

import (
	"encoding/json"
	"os"
	"server/diagnostics"
	"server/flow"
	"server/lt"
	sets "server/settings"
	"server/torr"
	"server/torr/state"
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
	cases := []struct {
		Kind    string `json:"kind"`
		Payload any    `json:"payload"`
	}{
		{"settings", sets.BTSets{CacheSize: 64 << 20, ReaderReadAHead: 95, PreloadCache: 50, Flow: sets.DefaultFlowSettings()}},
		{"settings", sets.BTSets{CacheSize: 64 << 20, Flow: nil}},
		{"torrent", state.TorrentStatus{Hash: hash, Title: "Generated fixture", Stat: state.TorrentInDB}},
		{"library", LibraryResponse{Items: []state.TorrentStatus{{Hash: hash, Title: "Fixture", Stat: state.TorrentInDB}}, Total: 1, LibraryTotal: 1, Page: 1, Limit: 50, Categories: []string{}, SampledAt: "2026-10-10T00:00:00Z"}},
		{"active", map[string]any{"items": []activeTorrent{{Torrent: state.TorrentStatus{Hash: hash, Title: "Fixture", Stat: state.TorrentWorking}, Status: compactFlowStatus{Hash: hash, Sessions: []torr.FlowSessionStatus{}}}}, "sampled_at": "2026-10-10T00:00:00Z"}},
		{"flow", FlowStatusResponse{Hash: hash, Sessions: []torr.FlowSessionStatus{{Group: "contract", FileIndex: 1, State: "PLAYING"}}}},
		{"flow", FlowStatusResponse{Hash: hash, Sessions: nil}},
		{"flow", FlowStatusResponse{Hash: hash, SampledAt: "2026-10-10T00:00:00Z", Timeline: &flow.TimelineSnapshot{Source: "server", Events: []flow.TimelineEvent{{Time: time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC), Type: "seek", ElapsedMs: 1000, OperationMs: 2, File: 1}}, Dropped: 0}, Sessions: []torr.FlowSessionStatus{{SessionID: "ABCDEFGHIJKLMNOPQRSTUV", FileIndex: 1, State: "PLAYING"}}}},
		{"flow", FlowStatusResponse{Hash: hash, StorageIO: lt.StorageIO{QueuedBytes: 16384, PeakQueueBytes: 32768, QueuedJobs: 2, CompletedJobs: 7, WaitP95Us: 128, CallbackP95Us: 256}, Sessions: []torr.FlowSessionStatus{{Group: "evidence", FileIndex: 1, State: "PLAYING", ReadableContiguousBytes: 32768, VerifiedContiguousBytes: 16384, Delivery: flow.DeliveryEvidence{DeficitBytes: 4096, DeficitSamples: 4}}}}},
		{"runtime", RuntimeStatus{BT: &torr.ClientStatusSnapshot{}, Memory: diagnostics.Memory()}},
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
