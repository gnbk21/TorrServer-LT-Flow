package flow

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// Summaries are derived from the existing opt-in bounded history. No second
// recorder, persistent media identifier or additional diagnostic opt-in exists.
type SessionSummary struct {
	Started              time.Time `json:"started"`
	LastObserved         time.Time `json:"last_observed"`
	Samples              uint64    `json:"samples"`
	BytesServed          int64     `json:"bytes_served"`
	PeakRecentReadStalls int64     `json:"peak_recent_read_stalls"`
	PeakRecentWaitP95Ms  float64   `json:"peak_recent_wait_p95_ms"`
	MinimumBufferBytes   int64     `json:"minimum_buffer_bytes"`
	MaximumBufferBytes   int64     `json:"maximum_buffer_bytes"`
	RangeRequests        uint64    `json:"range_requests"`
	RangeCancels         uint64    `json:"range_cancels"`
	Incomplete           bool      `json:"incomplete"`
}
type SessionSummaries struct {
	Enabled   bool             `json:"enabled"`
	Truncated bool             `json:"truncated"`
	Items     []SessionSummary `json:"items"`
}
type summaryKey struct {
	run              string
	torrent, session uint64
}

func ReadSessionSummaries(dir string, enabled bool) SessionSummaries {
	out := SessionSummaries{Enabled: enabled, Items: []SessionSummary{}}
	if !enabled {
		return out
	}
	rows := map[summaryKey]*SessionSummary{}
	order := []summaryKey{}
	closed := map[string]bool{}
	// Oldest first. At most four MiB and 64 sessions are retained for the UI.
	for archive := HistoryArchives; archive >= 0; archive-- {
		name := filepath.Join(dir, "flow-history.jsonl")
		if archive > 0 {
			name = fmt.Sprintf("%s.%d", name, archive)
		}
		info, err := os.Lstat(name)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil || !info.Mode().IsRegular() || info.Size() > HistoryFileBytes {
			out.Truncated = true
			continue
		}
		f, err := os.Open(name)
		if err != nil {
			out.Truncated = true
			continue
		}
		scanner := bufio.NewScanner(io.LimitReader(f, HistoryFileBytes))
		scanner.Buffer(make([]byte, HistoryRecordBytes), HistoryRecordBytes)
		for scanner.Scan() {
			var e HistoryEvent
			if json.Unmarshal(scanner.Bytes(), &e) != nil || !validHistoryEvent(e) || e.Run == "" {
				out.Truncated = true
				continue
			}
			if e.Type == "engine_stopped" {
				closed[e.Run] = true
			}
			if e.Dropped > 0 || e.Truncated {
				out.Truncated = true
			}
			var id uint64
			if e.Playback != nil {
				id = e.Playback.Session
			}
			if e.Response != nil {
				id = e.Response.Session
			}
			if id == 0 {
				continue
			}
			key := summaryKey{e.Run, e.Torrent, id}
			row := rows[key]
			if row == nil {
				if len(order) == 64 {
					delete(rows, order[0])
					order = order[1:]
					out.Truncated = true
				}
				row = &SessionSummary{Started: e.Time, LastObserved: e.Time, Incomplete: true}
				rows[key] = row
				order = append(order, key)
			}
			// Retain the most recently observed sessions, including an older
			// session that becomes active again after a newer session starts.
			for index, existing := range order {
				if existing == key {
					order = append(order[:index], order[index+1:]...)
					break
				}
			}
			order = append(order, key)
			row.LastObserved = e.Time
			if p := e.Playback; p != nil {
				if row.Samples == 0 {
					row.MinimumBufferBytes = max(0, p.BufferBytes)
				}
				row.Samples++
				row.MinimumBufferBytes = min(row.MinimumBufferBytes, max(0, p.BufferBytes))
				row.MaximumBufferBytes = max(row.MaximumBufferBytes, p.BufferBytes)
				row.PeakRecentReadStalls = max(row.PeakRecentReadStalls, p.RecentReadStalls)
				row.PeakRecentWaitP95Ms = max(row.PeakRecentWaitP95Ms, p.RecentWaitP95Ms)
				row.RangeRequests = max(row.RangeRequests, p.RangeRequests)
				row.RangeCancels = max(row.RangeCancels, p.RangeCancels)
				if p.State == "EXPIRED" {
					row.Incomplete = false
				}
			}
			if r := e.Response; r != nil {
				row.BytesServed += max(0, r.BytesServed)
			}
		}
		if scanner.Err() != nil {
			out.Truncated = true
		}
		_ = f.Close()
	}
	for i := len(order) - 1; i >= 0; i-- {
		key := order[i]
		row := *rows[key]
		if closed[key.run] {
			row.Incomplete = false
		}
		if out.Truncated {
			row.Incomplete = true
		}
		out.Items = append(out.Items, row)
	}
	return out
}
