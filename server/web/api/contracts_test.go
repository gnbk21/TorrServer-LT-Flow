package api

import (
	"encoding/json"
	"os"
	"server/diagnostics"
	sets "server/settings"
	"server/torr"
	"server/torr/state"
	"testing"
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
		{"flow", FlowStatusResponse{Hash: hash, Sessions: []torr.FlowSessionStatus{{Group: "contract", FileIndex: 1, State: "PLAYING"}}}},
		{"flow", FlowStatusResponse{Hash: hash, Sessions: nil}},
		{"runtime", RuntimeStatus{BT: &torr.ClientStatusSnapshot{}, Memory: diagnostics.Memory()}},
		{"network", torr.FlowNetworkStatus{State: "NO_ADDRESS", Connectivity: "INTERNET_WAIT"}},
	}
	data, err := json.MarshalIndent(cases, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(output, data, 0600); err != nil {
		t.Fatal(err)
	}
}
