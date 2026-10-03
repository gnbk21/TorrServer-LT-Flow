package flow

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestHistoryRetentionPrivacyAndRotation(t *testing.T) {
	dir := t.TempDir()
	name := filepath.Join(dir, "flow-history.jsonl")
	for i := 0; i <= HistoryArchives; i++ {
		path := name
		if i > 0 {
			path = fmt.Sprintf("%s.%d", name, i)
		}
		if err := os.WriteFile(path, bytes.Repeat([]byte("\n"), HistoryFileBytes-512), 0600); err != nil {
			t.Fatal(err)
		}
	}
	h, err := NewHistory(dir)
	if err != nil {
		t.Fatal(err)
	}
	h.Record(HistoryEvent{Type: "startup", Stage: "READY", Run: "secret-token", Torrent: 1, ElapsedMs: 223})
	h.Record(HistoryEvent{Type: "https://tracker/private?passkey=secret"})
	h.Record(HistoryEvent{Type: "startup", Stage: "token=private"})
	if !h.Close(time.Second) {
		t.Fatal("history did not drain")
	}
	if h.Status().Written != 1 || h.Status().Dropped != 2 {
		t.Fatalf("status: %+v", h.Status())
	}
	first, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(first), "secret") || strings.Contains(string(first), "private") {
		t.Fatal("untrusted strings retained")
	}
	var record HistoryEvent
	if err := json.Unmarshal(bytes.TrimSpace(first), &record); err != nil {
		t.Fatal(err)
	}
	if record.Run == "" || record.Time.Location() != time.UTC || record.ElapsedMs != 223 {
		t.Fatalf("record: %+v", record)
	}
	h2, err := NewHistory(dir)
	if err != nil {
		t.Fatal(err)
	}
	h2.Record(HistoryEvent{Type: "engine_started"})
	if !h2.Close(time.Second) {
		t.Fatal("restart did not drain")
	}
	second, _ := os.ReadFile(name)
	if !bytes.HasPrefix(second, first) {
		t.Fatal("restart lost existing history")
	}
	if h.run == h2.run {
		t.Fatal("restart reused run identity")
	}
	files, _ := os.ReadDir(dir)
	if len(files) != HistoryArchives+1 {
		t.Fatalf("unbounded files: %d", len(files))
	}
	for _, file := range files {
		st, _ := file.Info()
		if st.Size() > HistoryFileBytes {
			t.Fatalf("oversized %s", file.Name())
		}
	}
}

func TestHistoryQueueDoesNotBlockAndCloseIsBounded(t *testing.T) {
	// No consumer models a stalled OS write. Producers must still return and
	// account for overflow; shutdown must remain bounded too.
	h := &History{queue: make(chan HistoryEvent, HistoryQueueSize), stop: make(chan struct{}), done: make(chan struct{}), run: "fixture"}
	var wg sync.WaitGroup
	started := time.Now()
	for i := 0; i < 8; i++ {
		wg.Go(func() {
			for j := 0; j < 1000; j++ {
				h.Record(HistoryEvent{Type: "startup", Stage: "FIRST_BLOCK"})
			}
		})
	}
	wg.Wait()
	if time.Since(started) > time.Second || len(h.queue) != HistoryQueueSize || h.Status().Dropped != 8000-HistoryQueueSize {
		t.Fatalf("blocking/overflow: %+v", h.Status())
	}
	if h.Close(10 * time.Millisecond) {
		t.Fatal("stalled writer incorrectly completed")
	}
	h.Record(HistoryEvent{Type: "engine_started"}) // safe after stop; channel stays open
	close(h.done)
	if !h.Close(time.Second) {
		t.Fatal("completed writer not joined")
	}
}

func TestDHTFileBoundsAndAtomicReplacement(t *testing.T) {
	name := filepath.Join(t.TempDir(), "flow-dht.bin")
	for _, data := range [][]byte{[]byte("first"), []byte("second")} {
		if err := WriteDHTFile(name, data); err != nil {
			t.Fatal(err)
		}
		got, err := ReadDHTFile(name)
		if err != nil || !bytes.Equal(got, data) {
			t.Fatalf("replacement: %q %v", got, err)
		}
	}
	if err := WriteDHTFile(name, make([]byte, MaxDHTStateBytes+1)); err == nil {
		t.Fatal("accepted oversized state")
	}
	got, _ := ReadDHTFile(name)
	if string(got) != "second" {
		t.Fatal("failed write destroyed previous state")
	}
	files, _ := os.ReadDir(filepath.Dir(name))
	if len(files) != 1 {
		t.Fatal("temporary files leaked")
	}
	if err := os.WriteFile(name, make([]byte, MaxDHTStateBytes+1), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadDHTFile(name); err == nil {
		t.Fatal("unbounded read")
	}
}

func TestHistoryRepairsInterruptedLastRecord(t *testing.T) {
	dir := t.TempDir()
	name := filepath.Join(dir, "flow-history.jsonl")
	good := []byte(`{"type":"engine_started"}` + "\n")
	if err := os.WriteFile(name, append(append([]byte(nil), good...), []byte(`{"type":"start`)...), 0600); err != nil {
		t.Fatal(err)
	}
	h, err := NewHistory(dir)
	if err != nil {
		t.Fatal(err)
	}
	h.Record(HistoryEvent{Type: "engine_stopped"})
	if !h.Close(time.Second) {
		t.Fatal("history did not drain")
	}
	data, _ := os.ReadFile(name)
	if !bytes.HasPrefix(data, good) {
		t.Fatal("valid history was lost")
	}
	lines := bytes.Split(bytes.TrimSpace(data), []byte("\n"))
	if len(lines) != 2 {
		t.Fatalf("partial record retained: %q", data)
	}
	for _, line := range lines {
		if !json.Valid(line) {
			t.Fatalf("invalid history: %q", line)
		}
	}
}
