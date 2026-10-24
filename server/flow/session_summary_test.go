package flow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSessionSummariesBoundedPrivateAndRecovered(t *testing.T) {
	dir := t.TempDir()
	h, err := NewHistory(dir)
	if err != nil {
		t.Fatal(err)
	}
	h.Record(HistoryEvent{Type: "playback_sample", Torrent: 7, Playback: &PlaybackHistory{Session: 1, State: "PLAYING", WaitReason: "READY", BufferBytes: 100, RangeRequests: 3}})
	h.Record(HistoryEvent{Type: "response", Torrent: 7, Response: &ResponseHistory{Session: 1, Method: "GET", Classification: "PLAYBACK", BytesServed: 50}})
	if !h.Close(time.Second) {
		t.Fatal("writer did not drain")
	}
	rows := ReadSessionSummaries(dir, true)
	if len(rows.Items) != 1 || !rows.Items[0].Incomplete || rows.Items[0].BytesServed != 50 || rows.Items[0].Samples != 1 {
		t.Fatalf("wrong recovered evidence: %+v", rows)
	}
	encoded, _ := json.Marshal(rows)
	for _, key := range []string{"session\"", "torrent\"", "run\"", "hash", "token", "magnet"} {
		if strings.Contains(string(encoded), key) {
			t.Fatal("persistent identifier leaked", string(encoded))
		}
	}
	if len(ReadSessionSummaries(dir, false).Items) != 0 {
		t.Fatal("disabled history was read")
	}
	f, err := os.OpenFile(filepath.Join(dir, "flow-history.jsonl"), os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString("{broken\n")
	_ = f.Close()
	if !ReadSessionSummaries(dir, true).Truncated {
		t.Fatal("partial history claimed complete")
	}
}
