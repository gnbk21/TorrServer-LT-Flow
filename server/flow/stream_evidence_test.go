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

func TestStreamPolicyAggregatesRemainBoundedAndRetainable(t *testing.T) {
	s := &StreamEvidence{Known: true, RequestPolicy: &RequestPolicy{Enabled: true,
		SampledPeers: 512, QualifiedPeers: 512, CapacityRequests: 512 * 65535,
		DesiredRequests: 512 * 65535, QueuedRequests: 512 * 65535, HorizonMinMs: 250, HorizonMaxMs: 6000},
		RecoveryScan: &RecoveryScan{AgeMs: 20, Eligible: 1, UncertainRequest: 8191}}
	if !s.Valid() {
		t.Fatal("valid mixed recovery scan rejected")
	}
	data, err := json.Marshal(HistoryEvent{Type: "stream_sample", Stream: s})
	if err != nil || len(data) >= HistoryRecordBytes {
		t.Fatal("policy evidence exceeds private history record", len(data), err)
	}
	for _, mutate := range []func(*StreamEvidence){
		func(s *StreamEvidence) { s.RequestPolicy.HorizonMaxMs = 6001 },
		func(s *StreamEvidence) { s.RequestPolicy.QualifiedPeers = 513 },
		func(s *StreamEvidence) { s.RequestPolicy.DesiredRequests = s.RequestPolicy.CapacityRequests + 1 },
		func(s *StreamEvidence) { s.RecoveryScan.AgeMs = -2 },
		func(s *StreamEvidence) { s.RecoveryScan.NotLate = 1 },
	} {
		copy := *s
		policy, recovery := *s.RequestPolicy, *s.RecoveryScan
		copy.RequestPolicy, copy.RecoveryScan = &policy, &recovery
		mutate(&copy)
		if copy.Valid() {
			t.Fatal("unbounded policy evidence accepted")
		}
	}
}
