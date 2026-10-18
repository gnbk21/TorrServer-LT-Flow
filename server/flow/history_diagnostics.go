package flow

import (
	"bytes"
	"encoding/json"
	"os"
	"time"
)

// These payloads contain only bounded numeric evidence and validated enums.
// They describe server demand/delivery, never decoded player presentation.
type StartupHistory struct {
	Bootstrap         bool   `json:"bootstrap"`
	HeadBytes         int64  `json:"head_bytes"`
	HeadTargetBytes   int64  `json:"head_target_bytes"`
	HeadVerifiedBytes int64  `json:"head_verified_bytes"`
	TailBytes         int64  `json:"tail_bytes"`
	TailTargetBytes   int64  `json:"tail_target_bytes"`
	TailVerifiedBytes int64  `json:"tail_verified_bytes"`
	MissingHeadPieces int    `json:"missing_head_pieces"`
	MissingTailPieces int    `json:"missing_tail_pieces"`
	HeadBlockerPiece  *int   `json:"head_blocker_piece,omitempty"`
	TailBlockerPiece  *int   `json:"tail_blocker_piece,omitempty"`
	BlockedOn         string `json:"blocked_on"`
}

func (s StartupHistory) valid() bool {
	switch s.BlockedOn {
	case "HEAD", "INDEX", "HEAD_AND_INDEX", "PROBE", "NONE":
		return true
	}
	return false
}

type PlaybackHistory struct {
	Session              uint64  `json:"session"` // process-local sequence, not a client token
	State                string  `json:"state"`
	DemandOffsetBytes    int64   `json:"demand_offset_bytes"`
	DeliveredOffsetBytes int64   `json:"delivered_offset_bytes"`
	BufferBytes          int64   `json:"buffer_bytes"`
	BufferSeconds        float64 `json:"buffer_seconds"`
	DeliveryRate         float64 `json:"delivery_rate"`
	DownloadRate         float64 `json:"download_rate"`
	ConsumptionRate      float64 `json:"consumption_rate"`
	SupplyQualified      bool    `json:"supply_qualified"`
	WaitReason           string  `json:"wait_reason"`
	ActiveReaders        int     `json:"active_readers"`
	RangeRequests        uint64  `json:"range_requests"`
	RangeCancels         uint64  `json:"range_cancels"`
	RecentReadStalls     int64   `json:"recent_read_stalls"`
	RecentWaitP95Ms      float64 `json:"recent_wait_p95_ms"`
	RequiredPiece        int     `json:"required_piece"`
	RequiredPieceKnown   bool    `json:"required_piece_known"`
	Suppliers            *int    `json:"suppliers,omitempty"`
}

func (p PlaybackHistory) valid() bool {
	switch p.State {
	case "NEW", "PLAYING", "SEEK_RECOVERY", "WARM_IDLE", "EXPIRED":
	default:
		return false
	}
	switch p.WaitReason {
	case "", "READY", "METADATA", "UNKNOWN", "THROUGHPUT", "DISCOVERY", "CONNECTION", "MISSING_CONNECTED", "UNCHOKE", "FIRST_BLOCK":
		return true
	}
	return false
}

type peerHistoryKey struct {
	torrent         uint64
	code, operation int
	stage           string
}

type ResponseHistory struct {
	FirstByteObserved bool   `json:"first_byte_observed"`
	Session           uint64 `json:"session"`
	Method            string `json:"method"`
	Classification    string `json:"classification"`
	Status            int    `json:"status"`
	BytesServed       int64  `json:"bytes_served"`
	LifetimeMs        int64  `json:"lifetime_ms"`
	TTFBMs            int64  `json:"ttfb_ms"`
	Cancelled         bool   `json:"cancelled"`
}

func (r ResponseHistory) valid() bool {
	if r.Method != "GET" && r.Method != "HEAD" {
		return false
	}
	switch r.Classification {
	case "UNKNOWN", "HEAD_PROBE", "TAIL_INDEX", "SEEK", "HEADER_REREAD", "PLAYBACK":
		return true
	}
	return false
}

const peerHistoryGroups = 64

func (h *History) recordPeer(e HistoryEvent) {
	h.peerMu.Lock()
	defer h.peerMu.Unlock()
	if h.closed.Load() {
		return
	}
	if h.peers == nil {
		h.peers = make(map[peerHistoryKey]HistoryEvent)
	}
	key := peerHistoryKey{e.Torrent, e.Code, e.Operation, e.Stage}
	if _, ok := h.peers[key]; !ok && len(h.peers) >= peerHistoryGroups-1 {
		key = peerHistoryKey{stage: "OTHER"}
		e.Torrent, e.Code, e.Operation, e.Stage, e.Truncated = 0, 0, 0, "OTHER", true
	}
	row := h.peers[key]
	if row.Count == 0 {
		row = e
		row.Type = "peer_summary"
	}
	row.Count++
	row.WindowMs = max(int64(0), e.Time.Sub(row.Time).Milliseconds())
	h.peers[key] = row
}

func (h *History) flushPeers(now time.Time) []HistoryEvent {
	h.peerMu.Lock()
	defer h.peerMu.Unlock()
	out := make([]HistoryEvent, 0, len(h.peers))
	for _, row := range h.peers {
		row.WindowMs = max(row.WindowMs, now.Sub(row.Time).Milliseconds())
		out = append(out, row)
	}
	h.peers = nil
	return out
}

// Inspect the bounded current file, not just its last event: a final aggregated
// peer summary may follow engine_stopped. Missing closure is not proof of a crash.
func historyUnclosed(f *os.File, size int64) bool {
	if size == 0 {
		return false
	}
	buf := make([]byte, min(size, int64(HistoryFileBytes)))
	if _, err := f.ReadAt(buf, size-int64(len(buf))); err != nil {
		return false
	}
	lines := bytes.Split(bytes.TrimSpace(buf), []byte("\n"))
	latestRun := ""
	for i := len(lines) - 1; i >= 0; i-- {
		var event HistoryEvent
		if json.Unmarshal(lines[i], &event) != nil || event.Run == "" {
			continue
		}
		if latestRun == "" {
			latestRun = event.Run
		}
		if event.Run != latestRun {
			break
		}
		if event.Type == "engine_stopped" {
			return false
		}
	}
	return latestRun != ""
}

// Flush is reserved for process exit paths, never playback or native alerts.
// A stalled filesystem cannot delay those paths beyond wait.
func (h *History) Flush(wait time.Duration) bool {
	return h.flushEvent(nil, wait)
}

type historyFlush struct {
	event *HistoryEvent
	done  chan bool
}

func (h *History) RecordExit(reason string, wait time.Duration) bool {
	if h == nil {
		return true
	}
	switch reason {
	case "API_SHUTDOWN", "SIGNAL", "SERVICE_STOP", "SERVER_ERROR", "SHUTDOWN_TIMEOUT":
	default:
		return false
	}
	event := HistoryEvent{Type: "shutdown_requested", Stage: reason, Time: time.Now().UTC(), Run: h.run}
	return h.flushEvent(&event, wait)
}

func (h *History) flushEvent(event *HistoryEvent, wait time.Duration) bool {
	if h == nil {
		return true
	}
	timer := time.NewTimer(wait)
	defer timer.Stop()
	done := make(chan bool, 1) // caller may time out while the OS write finishes
	select {
	case h.flush <- historyFlush{event: event, done: done}:
	case <-h.done:
		return event == nil && h.failures.Load() == 0
	case <-timer.C:
		return false
	}
	select {
	case success := <-done:
		return success
	case <-timer.C:
		return false
	}
}
