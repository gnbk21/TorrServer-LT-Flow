package flow

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func readDiagnosticEvents(t *testing.T, dir string) []HistoryEvent {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "flow-history.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	var out []HistoryEvent
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line == "" {
			continue
		}
		var e HistoryEvent
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			t.Fatal(err)
		}
		out = append(out, e)
	}
	return out
}

func TestPeerStormPreservesCountsAndImportantEvents(t *testing.T) {
	dir := t.TempDir()
	h, err := NewHistory(dir)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 10000; i++ {
		h.Record(HistoryEvent{Type: "peer_disconnected", Torrent: 1, Code: 36, Operation: 16, Stage: "OTHER"})
	}
	h.Record(HistoryEvent{Type: "startup", Stage: "READY"})
	h.Record(HistoryEvent{Type: "engine_stopped"})
	closeHistoryFileForTest(t, h)
	events := readDiagnosticEvents(t, dir)
	var count uint64
	ready := false
	for _, e := range events {
		if e.Type == "peer_summary" {
			count += e.Count
		}
		ready = ready || e.Type == "startup"
	}
	if count != 10000 || !ready || h.Status().Dropped != 0 || len(events) != 3 {
		t.Fatalf("lost evidence: count=%d ready=%v status=%+v events=%d", count, ready, h.Status(), len(events))
	}
}

func TestPeerAggregationBoundsDistinctCodesWithoutLosingCounts(t *testing.T) {
	h := &History{}
	for i := 0; i < 1000; i++ {
		h.recordPeer(HistoryEvent{Type: "peer_disconnected", Torrent: uint64(i + 1), Code: i, Time: time.Now()})
	}
	events := h.flushPeers(time.Now())
	var count uint64
	truncated := false
	for _, e := range events {
		count += e.Count
		truncated = truncated || e.Truncated
	}
	if len(events) > peerHistoryGroups || count != 1000 || !truncated {
		t.Fatalf("unbounded/lost aggregation: groups=%d count=%d truncated=%v", len(events), count, truncated)
	}
	if len(h.flushPeers(time.Now())) != 0 {
		t.Fatal("counters were not reset")
	}
}

func TestPeerAggregationKeepsCategoryTransportAndFamilySeparate(t *testing.T) {
	h := &History{}
	for _, event := range []HistoryEvent{
		{ErrorCategory: "SYSTEM", Transport: "tcp", AddressFamily: "IPV4"},
		{ErrorCategory: "HTTP", Transport: "tcp", AddressFamily: "IPV4"},
		{ErrorCategory: "SYSTEM", Transport: "utp", AddressFamily: "IPV4"},
		{ErrorCategory: "SYSTEM", Transport: "tcp", AddressFamily: "IPV6"},
	} {
		event.Type, event.Torrent, event.Code, event.Operation = "peer_disconnected", 1, 60, 16
		h.recordPeer(event)
	}
	if events := h.flushPeers(time.Now()); len(events) != 4 {
		t.Fatal("distinct native failures were merged", events)
	}
}

func TestHistoryNestedEnumsCannotRetainSecrets(t *testing.T) {
	dir := t.TempDir()
	h, err := NewHistory(dir)
	if err != nil {
		t.Fatal(err)
	}
	h.Record(HistoryEvent{Type: "startup_sample", Startup: &StartupHistory{BlockedOn: "https://private?passkey=secret"}})
	h.Record(HistoryEvent{Type: "playback_sample", Playback: &PlaybackHistory{State: "PLAYING", WaitReason: "192.0.2.10"}})
	h.Record(HistoryEvent{Type: "startup_sample", Startup: &StartupHistory{BlockedOn: "INDEX", MissingTailPieces: 1}})
	h.Record(HistoryEvent{Type: "playback_sample", Playback: &PlaybackHistory{State: "PLAYING", WaitReason: "UNCHOKE", Session: 1}})
	closeHistoryFileForTest(t, h)
	if h.Status().Dropped != 2 || len(readDiagnosticEvents(t, dir)) != 2 {
		t.Fatalf("privacy validation: %+v", h.Status())
	}
}

func TestHistoryReportsUnconfirmedTerminationAndFlushesExitRequest(t *testing.T) {
	dir := t.TempDir()
	h, err := NewHistory(dir)
	if err != nil {
		t.Fatal(err)
	}
	h.Record(HistoryEvent{Type: "engine_started"})
	h.Record(HistoryEvent{Type: "shutdown_requested", Stage: "SIGNAL"})
	if !h.Flush(historyFileTestWait) {
		t.Fatal("exit request not flushed")
	}
	events := readDiagnosticEvents(t, dir)
	if len(events) != 2 || events[1].Stage != "SIGNAL" {
		t.Fatalf("flush did not persist exit intent: %+v", events)
	}
	closeHistoryFileForTest(t, h)
	h, err = NewHistory(dir)
	if err != nil {
		t.Fatal(err)
	}
	h.Record(HistoryEvent{Type: "engine_stopped"})
	closeHistoryFileForTest(t, h)
	events = readDiagnosticEvents(t, dir)
	if events[2].Type != "previous_run" || events[2].Stage != "UNCONFIRMED_TERMINATION" {
		t.Fatalf("missing unclean receipt: %+v", events)
	}
	h, err = NewHistory(dir)
	if err != nil {
		t.Fatal(err)
	}
	closeHistoryFileForTest(t, h)
	if len(readDiagnosticEvents(t, dir)) != len(events) {
		t.Fatal("clean closure mislabeled as unconfirmed")
	}
}

func TestHistoryFlushBoundedWithoutConsumer(t *testing.T) {
	h := &History{flush: make(chan historyFlush), done: make(chan struct{})}
	if h.Flush(time.Millisecond) {
		t.Fatal("stalled writer incorrectly flushed")
	}
}

func TestCleanEngineStopWithTrailingPeerSummaryIsNotCrash(t *testing.T) {
	dir := t.TempDir()
	data := []byte("{\"run\":\"previous\",\"type\":\"engine_stopped\"}\n{\"run\":\"previous\",\"type\":\"peer_summary\",\"count\":20}\n")
	if err := os.WriteFile(filepath.Join(dir, "flow-history.jsonl"), data, 0600); err != nil {
		t.Fatal(err)
	}
	h, err := NewHistory(dir)
	if err != nil {
		t.Fatal(err)
	}
	closeHistoryFileForTest(t, h)
	if h.Status().Written != 0 {
		t.Fatal("trailing peer summary caused false crash receipt")
	}
}

func TestExitEvidenceBypassesFullOrdinaryQueue(t *testing.T) {
	dir := t.TempDir()
	name := filepath.Join(dir, "flow-history.jsonl")
	f, size, err := openHistory(name)
	if err != nil {
		t.Fatal(err)
	}
	h := &History{queue: make(chan HistoryEvent, HistoryQueueSize), stop: make(chan struct{}), done: make(chan struct{}), flush: make(chan historyFlush), run: "fixture"}
	for i := 0; i < HistoryQueueSize; i++ {
		h.Record(HistoryEvent{Type: "first_byte"})
	}
	h.Record(HistoryEvent{Type: "first_byte"}) // prove the ordinary queue is full
	go h.writeLoop(f, name, size)
	if !h.RecordExit("SIGNAL", historyFileTestWait) {
		t.Fatal("full queue discarded exit evidence")
	}
	closeHistoryFileForTest(t, h)
	found := false
	for _, e := range readDiagnosticEvents(t, dir) {
		found = found || e.Type == "shutdown_requested" && e.Stage == "SIGNAL"
	}
	if !found || h.Status().Dropped != 1 {
		t.Fatalf("lost exit intent: %+v", h.Status())
	}
}

func TestHistoryActualEncodedRecordLimitKeepsRotationBounded(t *testing.T) {
	dir := t.TempDir()
	h, err := NewHistory(dir)
	if err != nil {
		t.Fatal(err)
	}
	e := HistoryEvent{Type: "playback_sample", Playback: &PlaybackHistory{State: "PLAYING", WaitReason: "READY", Session: 1, DownloadRate: 1e10, BufferSeconds: 120}}
	for i := 0; i < 1800; i++ {
		h.Record(e)
		if i%100 == 0 && !h.Flush(historyFileTestWait) {
			t.Fatal("flush")
		}
	}
	closeHistoryFileForTest(t, h)
	for i := 0; i <= HistoryArchives; i++ {
		name := filepath.Join(dir, "flow-history.jsonl")
		if i > 0 {
			name = fmt.Sprintf("%s.%d", name, i)
		}
		data, err := os.ReadFile(name)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		if len(data) > HistoryFileBytes {
			t.Fatal("file retention bound exceeded")
		}
		for _, line := range bytes.Split(bytes.TrimSpace(data), []byte("\n")) {
			if len(line) > HistoryRecordBytes || !json.Valid(line) {
				t.Fatal("invalid/oversized line")
			}
		}
	}
}
