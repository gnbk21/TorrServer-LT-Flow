package torr

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"server/flow"
	"server/lt"
	"server/settings"
	"server/torr/storage/torrstor"
	"strings"
	"testing"
	"time"
)

func TestPeriodicPlaybackHistoryAndResponsePrivacy(t *testing.T) {
	old := settings.BTsets()
	t.Cleanup(func() { settings.StoreBTsets(old) })
	f := settings.DefaultFlowSettings()
	f.Enabled = false
	f.DiagnosticHistory = true
	settings.StoreBTsets(&settings.BTSets{Flow: f})
	dir := t.TempDir()
	h, err := flow.NewHistory(dir)
	if err != nil {
		t.Fatal(err)
	}
	bt := NewBTS()
	bt.history.Store(h)
	tor := &Torrent{TorrentSpec: &TorrentSpec{InfoHash: Hash{1}}, bt: bt, diagnosticID: 1, flowAddedAt: time.Now()}
	file := &File{Index: 0, Path: "private-movie.mkv", Length: 256 << 20}
	tor.lh.Store(&lt.Torrent{})
	tor.filesSnapshot.Store(&torrentFiles{sorted: []*File{file}, byID: map[int]*File{1: file}})
	req := httptest.NewRequest("GET", "/stream?token=secret", nil)
	_, _, seq := tor.flowStart(1, file, "ip:192.0.2.10", req)
	tor.flowBodyStart(1, "ip:192.0.2.10", seq, 32<<20)
	tor.flowProgress(1, "ip:192.0.2.10", seq, 36<<20)
	tor.flowStart(1, file, torrstor.ProbeReaderGroup, req)
	tor.recordPlaybackHistory(time.Now())
	tor.recordPlaybackHistory(time.Now())
	tor.flowEnd(1, "ip:192.0.2.10", seq, FlowRangeTrace{Method: "GET", Status: 206, BytesServed: 4 << 20, Cancelled: true, Classification: "HEADER_REREAD", TTFBMs: 25})
	if !h.Close(time.Second) {
		t.Fatal("history drain")
	}
	data, err := os.ReadFile(filepath.Join(dir, "flow-history.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"private-movie", "192.0.2.10", "secret", "probe:loopback"} {
		if strings.Contains(string(data), secret) {
			t.Fatal("private identity retained")
		}
	}
	samples, responses := 0, 0
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		var event flow.HistoryEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatal(err)
		}
		if event.Playback != nil {
			samples++
			if event.Playback.DeliveredOffsetBytes != 36<<20 || event.Playback.Session == 0 {
				t.Fatal("delivery evidence missing")
			}
		}
		if event.Response != nil {
			responses++
			if !event.Response.Cancelled || event.Response.Classification != "HEADER_REREAD" {
				t.Fatal("response outcome missing")
			}
		}
	}
	if samples != 1 || responses != 1 {
		t.Fatalf("cadence/internal-probe exclusion: samples=%d responses=%d", samples, responses)
	}
}

func TestStartupSnapshotDoesNotExposeMutablePointers(t *testing.T) {
	piece := 0
	seconds := 10.0
	tor := &Torrent{flowStartup: FlowStartupStatus{
		Progress: &flow.StartupHistory{BlockedOn: "HEAD", HeadBlockerPiece: &piece},
		Reserve:  &flow.ReservePlan{Known: true, TargetBytes: 64 << 20, AdditionalFillSeconds: &seconds},
	}}
	s := tor.FlowStartup()
	s.Progress.BlockedOn = "NONE"
	*s.Progress.HeadBlockerPiece = 10
	s.Reserve.TargetBytes = 0
	*s.Reserve.AdditionalFillSeconds = 0
	if tor.FlowStartup().Progress.BlockedOn != "HEAD" || *tor.FlowStartup().Progress.HeadBlockerPiece != 0 {
		t.Fatal("caller mutated internal progress")
	}
	if tor.FlowStartup().Reserve.TargetBytes != 64<<20 || *tor.FlowStartup().Reserve.AdditionalFillSeconds != 10 {
		t.Fatal("caller mutated the recorded startup decision")
	}
}

func TestStartupReserveReportsTheDecisionThatEnlargedTheGate(t *testing.T) {
	tor := &Torrent{}
	tor.recordStartupReserve(flow.ReservePlan{Reason: "UNKNOWN", TargetBytes: 32 << 20}, true)
	chosen := flow.ReservePlan{Known: true, Reason: "SUPPLY_DEFICIT", TargetBytes: 96 << 20}
	tor.recordStartupReserve(chosen, true)
	for _, filling := range []bool{true, false} {
		tor.recordStartupReserve(flow.ReservePlan{Known: true, Reason: "SUFFICIENT_SUPPLY", TargetBytes: 32 << 20}, filling)
		tor.recordStartupReserve(flow.ReservePlan{Reason: "UNKNOWN", TargetBytes: 32 << 20}, filling)
		if got := tor.FlowStartup().Reserve; got.TargetBytes != chosen.TargetBytes || got.Reason != chosen.Reason {
			t.Fatal("diagnostics no longer describe the retained gate", got)
		}
	}
	chosen.TargetBytes = 128 << 20
	tor.recordStartupReserve(chosen, true)
	if tor.FlowStartup().Reserve.TargetBytes != chosen.TargetBytes {
		t.Fatal("larger reserve was not recorded")
	}
}
