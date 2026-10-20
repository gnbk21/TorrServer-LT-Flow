package flow

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

const HistoryQueueSize = 256
const HistoryFileBytes = 1 << 20
const HistoryArchives = 3 // Four files total, at most 4 MiB.
const HistoryRecordBytes = 4096

// HistoryEvent deliberately has no names, hashes, URLs, IPs, free-text errors,
// request headers or arbitrary metadata. Strings are checked against enums.
// Torrent is a process-local sequence, not a persistent media identity.
type HistoryEvent struct {
	Startup           *StartupHistory  `json:"startup,omitempty"`
	Playback          *PlaybackHistory `json:"playback,omitempty"`
	Response          *ResponseHistory `json:"response,omitempty"`
	Count             uint64           `json:"count,omitempty"`
	WindowMs          int64            `json:"window_ms,omitempty"`
	Truncated         bool             `json:"truncated,omitempty"`
	Sparse            *SparseHistory   `json:"sparse,omitempty"`
	Time              time.Time        `json:"time"`
	Run               string           `json:"run"`
	Type              string           `json:"type"`
	Stage             string           `json:"stage,omitempty"`
	Torrent           uint64           `json:"torrent,omitempty"`
	File              int              `json:"file,omitempty"`
	ElapsedMs         int64            `json:"elapsed_ms"`
	OperationMeasured bool             `json:"operation_measured,omitempty"`
	Bytes             int64            `json:"bytes,omitempty"`
	Code              int              `json:"code,omitempty"`
	Operation         int              `json:"operation,omitempty"`
	Dropped           uint64           `json:"dropped,omitempty"`
}

type SparseHistory struct {
	UrgentKnown          bool   `json:"urgent_known,omitempty"`
	UrgentTruncated      bool   `json:"urgent_truncated,omitempty"`
	UrgentPiece          int    `json:"urgent_piece,omitempty"`
	UrgentPriority       int    `json:"urgent_priority,omitempty"`
	UrgentUnrequested    int    `json:"urgent_unrequested,omitempty"`
	UrgentRequested      int    `json:"urgent_requested,omitempty"`
	UrgentWriting        int    `json:"urgent_writing,omitempty"`
	UrgentFinished       int    `json:"urgent_finished,omitempty"`
	UrgentReceivingBytes int    `json:"urgent_receiving_bytes,omitempty"`
	UrgentDuplicates     int    `json:"urgent_duplicates,omitempty"`
	SampledPeers         int    `json:"sampled_peers"`
	Truncated            bool   `json:"truncated"`
	UsefulPeers          int    `json:"useful_peers"`
	ChokedPeers          int    `json:"choked_peers"`
	SnubbedPeers         int    `json:"snubbed_peers"`
	OutstandingBytes     int64  `json:"outstanding_bytes"`
	MaxQueueMs           int64  `json:"max_queue_ms"`
	FailedBytes          int64  `json:"failed_bytes"`
	RedundantBytes       int64  `json:"redundant_bytes"`
	RequestTimeouts      uint64 `json:"request_timeouts"`
	RequestsDropped      uint64 `json:"requests_dropped"`
}

type HistoryStatus struct {
	Enabled bool   `json:"enabled"`
	Written uint64 `json:"written"`
	Dropped uint64 `json:"dropped"`
	Errors  uint64 `json:"errors"`
}

type History struct {
	peerMu   sync.Mutex
	peers    map[peerHistoryKey]HistoryEvent
	queue    chan HistoryEvent
	stop     chan struct{}
	done     chan struct{}
	closed   atomic.Bool
	written  atomic.Uint64
	dropped  atomic.Uint64
	failures atomic.Uint64
	run      string
	flush    chan historyFlush
}

func NewHistory(dir string) (*History, error) {
	name := filepath.Join(dir, "flow-history.jsonl")
	for i := 1; i <= HistoryArchives; i++ {
		path := fmt.Sprintf("%s.%d", name, i)
		if st, err := os.Lstat(path); err == nil {
			if !st.Mode().IsRegular() {
				return nil, errors.New("history archive is not regular")
			}
			if st.Size() > HistoryFileBytes {
				if err := os.Truncate(path, 0); err != nil {
					return nil, err
				}
			}
		} else if !os.IsNotExist(err) {
			return nil, err
		}
	}
	f, size, err := openHistory(name)
	if err != nil {
		return nil, err
	}
	var run [8]byte
	if _, err := rand.Read(run[:]); err != nil {
		f.Close()
		return nil, err
	}
	previousUnclosed := historyUnclosed(f, size)
	h := &History{queue: make(chan HistoryEvent, HistoryQueueSize), stop: make(chan struct{}), done: make(chan struct{}), flush: make(chan historyFlush), run: hex.EncodeToString(run[:])}
	go h.writeLoop(f, name, size)
	if previousUnclosed {
		h.Record(HistoryEvent{Type: "previous_run", Stage: "UNCONFIRMED_TERMINATION"})
	}
	return h, nil
}

func validHistoryEvent(e HistoryEvent) bool {
	switch e.Type {
	case "engine_started", "engine_stopped", "shutdown_requested", "previous_run", "metadata", "metadata_recovery", "startup", "startup_sample", "playback_sample", "response", "peer_summary", "first_byte", "first_block", "probe", "dht_peer", "peer_disconnected", "dht", "sparse", "seek", "recovery", "storage":
	default:
		return false
	}
	switch e.Stage {
	case "METADATA_WAIT", "METADATA_HTTP", "METADATA_RECOVERED", "METADATA_PEERS", "METADATA_MISS":
	case "", "METADATA", "PEERS", "FIRST_BLOCK", "HEAD_INDEX", "PREBUFFER", "PROBE_GRACE", "READY", "PLAYING", "READER_WINDOW", "CANCELLED", "TIMEOUT", "FAILED", "DHT_RESTORED", "DHT_SAVED", "DHT_IGNORED", "PAUSED", "REDUNDANT", "OTHER", "CACHE_READ_SLOW", "CACHE_WRITE_SLOW", "CACHE_READ_FAILED", "CACHE_WRITE_FAILED", "API_SHUTDOWN", "SIGNAL", "SERVICE_STOP", "SERVER_ERROR", "SHUTDOWN_TIMEOUT", "UNCONFIRMED_TERMINATION":
	default:
		return false
	}
	if e.Startup != nil && !e.Startup.valid() || e.Playback != nil && !e.Playback.valid() || e.Response != nil && !e.Response.valid() {
		return false
	}
	return true
}

// Record never waits for disk or for queue space. It is safe during Close and
// after Close; the producer channel is never closed.
func (h *History) Record(e HistoryEvent) {
	if h == nil || h.closed.Load() {
		return
	}
	if !validHistoryEvent(e) {
		h.dropped.Add(1)
		return
	}
	if e.Time.IsZero() {
		e.Time = time.Now().UTC()
	}
	e.Run = h.run
	if e.Type == "peer_disconnected" {
		h.recordPeer(e)
		return
	}
	select {
	case h.queue <- e:
	default:
		h.dropped.Add(1)
	}
}

func (h *History) Status() HistoryStatus {
	if h == nil {
		return HistoryStatus{}
	}
	return HistoryStatus{!h.closed.Load(), h.written.Load(), h.dropped.Load(), h.failures.Load()}
}

// Close requests a finite queue drain. A stalled filesystem cannot hold up
// engine shutdown beyond wait. The caller must not reopen the same history
// until Done closes (the previous writer can still be completing an OS write).
func (h *History) Close(wait time.Duration) bool {
	if h == nil {
		return true
	}
	if h.closed.CompareAndSwap(false, true) {
		close(h.stop)
	}
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-h.done:
		return true
	case <-timer.C:
		return false
	}
}

func openHistory(name string) (*os.File, int64, error) {
	if st, err := os.Lstat(name); err == nil && !st.Mode().IsRegular() {
		return nil, 0, errors.New("history is not a regular file")
	} else if err != nil && !os.IsNotExist(err) {
		return nil, 0, err
	}
	f, err := os.OpenFile(name, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, 0, err
	}
	st, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, 0, err
	}
	if st.Size() > HistoryFileBytes {
		if err := f.Truncate(0); err != nil {
			f.Close()
			return nil, 0, err
		}
		return f, 0, nil
	}
	size := st.Size()
	if size > 0 {
		// A killed process or short disk write may leave one partial line. Every
		// event is bounded, so recovery never scans the entire retained file.
		n := min(size, int64(HistoryRecordBytes))
		tail := make([]byte, n)
		if _, err := f.ReadAt(tail, size-n); err != nil {
			f.Close()
			return nil, 0, err
		}
		if tail[len(tail)-1] != '\n' {
			i := bytes.LastIndexByte(tail, '\n')
			if i < 0 {
				size = 0
			} else {
				size = size - n + int64(i) + 1
			}
			if err := f.Truncate(size); err != nil {
				f.Close()
				return nil, 0, err
			}
		}
	}
	if _, err := f.Seek(size, io.SeekStart); err != nil {
		f.Close()
		return nil, 0, err
	}
	return f, size, nil
}

func rotateHistory(name string) error {
	// Only these four exact file names are managed, never directory contents.
	for i := HistoryArchives; i >= 1; i-- {
		dst := fmt.Sprintf("%s.%d", name, i)
		src := name
		if i > 1 {
			src = fmt.Sprintf("%s.%d", name, i-1)
		}
		if err := os.Remove(dst); err != nil && !os.IsNotExist(err) {
			return err
		}
		if err := os.Rename(src, dst); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func (h *History) consume(w io.Writer, encoded []byte) int64 {
	b := append(encoded, '\n')
	n, err := w.Write(b)
	if err != nil || n != len(b) {
		h.failures.Add(1)
	} else {
		h.written.Add(1)
	}
	return int64(n)
}

func (h *History) writeLoop(f *os.File, name string, size int64) {
	defer close(h.done)
	defer func() {
		if f != nil {
			_ = f.Close()
		}
	}()
	write := func(e HistoryEvent) {
		// Size the actual record: diagnostic snapshots may be larger than old
		// scalar events, but no record may exceed the bounded recovery window.
		e.Dropped = h.dropped.Load()
		encoded, err := json.Marshal(e)
		if err != nil || len(encoded)+1 > HistoryRecordBytes {
			h.failures.Add(1)
			return
		}
		if f == nil {
			h.failures.Add(1)
			return
		}
		if size+int64(len(encoded))+1 > HistoryFileBytes {
			_ = f.Close()
			f = nil
			if err := rotateHistory(name); err != nil {
				h.failures.Add(1)
				return
			}
			var err error
			f, size, err = openHistory(name)
			if err != nil {
				h.failures.Add(1)
				return
			}
		}
		if f == nil {
			h.failures.Add(1)
			return
		}
		beforeErrors := h.failures.Load()
		size += h.consume(f, encoded)
		if h.failures.Load() != beforeErrors {
			// Do not append complete records to a partial line after disk failure.
			// Producers keep returning immediately; a restart repairs the tail.
			_ = f.Close()
			f = nil
		}
	}
	peerTick := time.NewTicker(10 * time.Second)
	defer peerTick.Stop()
	for {
		select {
		case e := <-h.queue:
			write(e)
		case now := <-peerTick.C:
			for _, event := range h.flushPeers(now) {
				write(event)
			}
		case request := <-h.flush:
			beforeErrors := h.failures.Load()
			for i := 0; i < HistoryQueueSize; i++ {
				select {
				case event := <-h.queue:
					write(event)
				default:
					i = HistoryQueueSize
				}
			}
			// Exit intent has a separate bounded handshake, so ordinary queue
			// overflow cannot discard the reason we are about to terminate.
			if request.event != nil {
				write(*request.event)
			}
			if f != nil {
				if err := f.Sync(); err != nil {
					h.failures.Add(1)
				}
			}
			request.done <- f != nil && h.failures.Load() == beforeErrors
		case <-h.stop:
			defer func() {
				if f != nil {
					if err := f.Sync(); err != nil {
						h.failures.Add(1)
					}
				}
			}()
			for _, event := range h.flushPeers(time.Now()) {
				write(event)
			}
			for i := 0; i < HistoryQueueSize; i++ {
				select {
				case e := <-h.queue:
					write(e)
				default:
					return
				}
			}
			return
		}
	}
}
