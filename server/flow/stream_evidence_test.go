package flow

import (
	"encoding/json"
	"testing"
)

func TestStartupPhasesKeepUnknownSeparateFromZeroAndDifferentClocks(t *testing.T) {
	rows := StartupPhases(nil, 0, -1, -1, -1, -1, -1, true)
	for _, i := range []int{0, 1, 2, 4, 5, 7, 8} {
		if rows[i].DurationMs != nil {
			t.Fatalf("phase %d invented timing: %+v", i, rows[i])
		}
	}
	if *rows[3].DurationMs != 0 || *rows[6].DurationMs != 0 {
		t.Fatal("known local metadata/cached probe lost")
	}
	n := &StreamEvidence{Known: true, FirstDialMs: 20, ConnectDurationMs: 10, HandshakeDurationMs: 50, FirstPayloadMs: 90}
	rows = StartupPhases(n, 100, 200, 205, 230, 300, 310, false)
	if *rows[6].DurationMs != 25 || *rows[7].DurationMs != 300 || rows[0].Basis == rows[7].Basis {
		t.Fatal("overlapping clocks were subtracted or conflated")
	}
	rows = StartupPhases(n, 100, -1, 230, 205, -1, -1, false)
	if rows[6].DurationMs != nil {
		t.Fatal("inverted probe duration accepted")
	}
}

func TestStreamHistoryRejectsUnboundedOrIdentifyingGroups(t *testing.T) {
	s := &StreamEvidence{Known: true, Service: []StreamService{{Transport: "tcp", AddressFamily: "IPV4", P50Ms: 100, P95Ms: 250}}, PeakHedges: 16}
	if !s.Valid() {
		t.Fatal("valid aggregate rejected")
	}
	for _, mutate := range []func(*StreamEvidence){
		func(s *StreamEvidence) { s.Service = make([]StreamService, 7) },
		func(s *StreamEvidence) { s.ActiveHedges = 17 },
		func(s *StreamEvidence) { s.ActiveHedgeBytes = 256<<10 + 1 },
		func(s *StreamEvidence) { s.Frontiers = 9 },
		func(s *StreamEvidence) { s.FirstHandshakeMs = -2 },
		func(s *StreamEvidence) { s.ObsoleteBytes = 1 },
		func(s *StreamEvidence) { s.Service = []StreamService{{Transport: "192.0.2.1", AddressFamily: "IPV4"}} },
	} {
		copy := *s
		mutate(&copy)
		if copy.Valid() {
			t.Fatal("invalid aggregate accepted")
		}
	}
	e := HistoryEvent{Type: "stream_sample", Stream: s}
	data, err := json.Marshal(e)
	if err != nil || len(data) >= HistoryRecordBytes || !validHistoryEvent(e) {
		t.Fatal("stream history cannot be retained")
	}
}
