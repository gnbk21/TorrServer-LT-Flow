package torr

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"server/flow"
	"server/log"
	"server/settings"
	"server/torr/storage/torrstor"
)

type FlowRangeTrace struct {
	Timestamp      time.Time `json:"timestamp"`
	Group          string    `json:"group"`
	Method         string    `json:"method"`
	Start          int64     `json:"start"`
	End            int64     `json:"end"`
	Status         int       `json:"status"`
	BytesServed    int64     `json:"bytes_served"`
	LifetimeMs     int64     `json:"lifetime_ms"`
	TTFBMs         int64     `json:"ttfb_ms"`
	Cancelled      bool      `json:"cancelled"`
	Classification string    `json:"classification"`
}

type FlowStartupStatus struct {
	FileIndex                int    `json:"file_index"`
	State                    string `json:"state"`
	BootstrapHeadTargetBytes int64  `json:"bootstrap_head_target_bytes"`
	BootstrapTailTargetBytes int64  `json:"bootstrap_tail_target_bytes"`
	StartupTargetBytes       int64  `json:"startup_target_bytes"`
	BootstrapCompleteMs      int64  `json:"bootstrap_complete_ms"`
	ProbeStartMs             int64  `json:"probe_start_ms"`
	ProbeCompleteMs          int64  `json:"probe_complete_ms"`
	ProbeSuccess             bool   `json:"probe_success"`
	StartupPrebufferMs       int64  `json:"startup_prebuffer_ms"`
}

func (t *Torrent) FlowStartup() FlowStartupStatus {
	if t == nil {
		return FlowStartupStatus{}
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.flowStartup
}

type FlowSessionStatus struct {
	flow.CounterSnapshot
	Group                     string           `json:"group"`
	FileIndex                 int              `json:"file_index"`
	FileSize                  int64            `json:"file_size"`
	State                     string           `json:"state"`
	ActiveReaders             int              `json:"active_readers"`
	PlaybackOffsetBytes       int64            `json:"playback_offset_bytes"`
	PlaybackOffsetSeconds     float64          `json:"playback_offset_seconds"`
	BufferAheadBytes          int64            `json:"buffer_ahead_bytes"`
	BufferAheadSeconds        float64          `json:"buffer_ahead_seconds"`
	EstimatedMediaBitrate     float64          `json:"estimated_media_bitrate"`
	BitrateEstimateSource     string           `json:"bitrate_estimate_source"`
	BitrateEstimateConfidence string           `json:"bitrate_estimate_confidence"`
	DownloadRate              float64          `json:"download_rate"`
	UploadRate                float64          `json:"upload_rate"`
	SustainabilityRatio       float64          `json:"sustainability_ratio"`
	CacheUsed                 int64            `json:"cache_used"`
	CacheSize                 int64            `json:"cache_size"`
	ConnectedPeers            int              `json:"connected_peers"`
	RangeRequestCount         uint64           `json:"range_request_count"`
	RangeCancelCount          uint64           `json:"range_cancel_count"`
	SeekCount                 uint64           `json:"seek_count"`
	WarmReconnectCount        uint64           `json:"warm_reconnect_count"`
	LastClassification        string           `json:"last_classification"`
	LastTTFBMs                int64            `json:"last_ttfb_ms"`
	Traces                    []FlowRangeTrace `json:"traces,omitempty"`
}

type flowSession struct {
	FlowSessionStatus
	fileOffset int64
	lastSeen   time.Time
}

// flowStart records a logical playback session separately from a TCP request.
func (t *Torrent) flowStart(fileID int, file *File, group string, req *http.Request) (string, flow.RangeHint) {
	hint := flow.ParseRangeHint(req.Header.Get("Range"), file.Length)
	if t == nil {
		return "UNKNOWN", hint
	}
	internal := group == torrstor.ProbeReaderGroup
	if !internal && req.Method == http.MethodGet && settings.CurrentFlow().Enabled {
		if cache := torrstor.Global().CacheByHash([20]byte(t.Hash())); cache != nil {
			cache.ClearWarmReserve()
		}
		t.mu.Lock()
		t.warmIdleSince = time.Time{}
		if t.flowStartup.FileIndex == fileID {
			t.flowStartup.State = "PLAYING"
		}
		t.mu.Unlock()
	}
	if !settings.CurrentFlow().MetricsEnabled {
		return "UNKNOWN", hint
	}
	key := fmt.Sprintf("%d/%s", fileID, group)
	t.flowMu.Lock()
	defer t.flowMu.Unlock()
	if t.flowSessions == nil {
		t.flowSessions = make(map[string]*flowSession)
	}
	// A torrent lifetime bounds the map, and this cap bounds hostile unique tokens.
	if len(t.flowSessions) >= 128 && t.flowSessions[key] == nil {
		var oldest string
		var at time.Time
		for k, s := range t.flowSessions {
			if oldest == "" || s.lastSeen.Before(at) {
				oldest, at = k, s.lastSeen
			}
		}
		delete(t.flowSessions, oldest)
	}
	s := t.flowSessions[key]
	if s == nil {
		s = &flowSession{FlowSessionStatus: FlowSessionStatus{Group: group, FileIndex: fileID, FileSize: file.Length, State: "NEW"}, fileOffset: file.Offset}
		t.flowSessions[key] = s
	}
	classification := flow.Classify(req.Method, internal, hint, file.Length, s.PlaybackOffsetBytes, s.RangeRequestCount > 0)
	if !settings.CurrentFlow().RangeClassification {
		classification = "UNKNOWN"
	}
	if !internal {
		if s.State == "WARM_IDLE" {
			s.WarmReconnectCount++
		}
		if classification == "SEEK" {
			s.SeekCount++
			s.State = "SEEK_RECOVERY"
		} else {
			s.State = "PLAYING"
		}
		s.ActiveReaders++
		if hint.Valid {
			s.PlaybackOffsetBytes = hint.Start
		}
	}
	s.RangeRequestCount++
	s.LastClassification = classification
	s.lastSeen = time.Now()
	return classification, hint
}

func (t *Torrent) flowReaderClosed(file *File, offset int64) {
	if t == nil || !settings.CurrentFlow().Enabled {
		return
	}
	if cache := torrstor.Global().CacheByHash([20]byte(t.Hash())); cache != nil && cache.StreamingReaders() == 0 {
		t.mu.Lock()
		started := t.warmIdleSince.IsZero() && cache.StreamingReaders() == 0
		if started {
			t.warmIdleSince = time.Now()
		}
		t.mu.Unlock()
		if started && file != nil && file.Length > 0 && cache.PieceLength > 0 {
			position := file.Offset + offset
			if position < file.Offset {
				position = file.Offset
			}
			if position >= file.Offset+file.Length {
				position = file.Offset + file.Length - 1
			}
			piece := int(position / cache.PieceLength)
			fileFirst := int(file.Offset / cache.PieceLength)
			fileLast := int((file.Offset + file.Length - 1) / cache.PieceLength)
			budget := int64(32 << 20)
			if s := settings.BTsets(); s != nil && s.CacheSize > 0 {
				budget = min(budget, s.CacheSize/4)
			}
			ahead := int(max(int64(1), budget/cache.PieceLength))
			cache.SetWarmReserve(max(fileFirst, piece-2), min(fileLast, piece+ahead), torrentExpireTimeout())
		}
	}
}

func (t *Torrent) flowEnd(fileID int, group string, tr FlowRangeTrace, hint flow.RangeHint) {
	if t == nil || !settings.CurrentFlow().MetricsEnabled {
		return
	}
	key := fmt.Sprintf("%d/%s", fileID, group)
	t.flowMu.Lock()
	defer t.flowMu.Unlock()
	s := t.flowSessions[key]
	if s == nil {
		return
	}
	if tr.Cancelled {
		s.RangeCancelCount++
	}
	if group != torrstor.ProbeReaderGroup {
		if s.ActiveReaders > 0 {
			s.ActiveReaders--
		}
		if hint.Valid && tr.BytesServed > 0 {
			s.PlaybackOffsetBytes = hint.Start + tr.BytesServed
		}
		if s.ActiveReaders == 0 {
			s.State = "WARM_IDLE"
		} else {
			s.State = "PLAYING"
		}
	}
	s.LastTTFBMs = tr.TTFBMs
	s.lastSeen = time.Now()
	if settings.CurrentFlow().RangeTraceEnabled {
		if len(s.Traces) >= 128 {
			copy(s.Traces, s.Traces[1:])
			s.Traces = s.Traces[:127]
		}
		s.Traces = append(s.Traces, tr)
		if settings.CurrentFlow().DebugFlow {
			if b, err := json.Marshal(tr); err == nil {
				log.TLogln("FLOW RANGE", string(b))
			}
		}
	}
}

// FlowStatus returns a bounded diagnostic snapshot. The cache remains the
// authority for resident bytes, so buffer seconds stop at the first hole.
func (t *Torrent) FlowStatus() []FlowSessionStatus {
	if t == nil {
		return nil
	}
	t.flowMu.Lock()
	out := make([]FlowSessionStatus, 0, len(t.flowSessions))
	var offsets []int64
	for _, s := range t.flowSessions {
		copy := s.FlowSessionStatus
		copy.Traces = append([]FlowRangeTrace(nil), s.Traces...)
		if copy.ActiveReaders == 0 && time.Since(s.lastSeen) >= time.Duration(settings.CurrentFlow().WarmSessionTimeoutSec)*time.Second {
			copy.State = "EXPIRED"
		}
		out = append(out, copy)
		offsets = append(offsets, s.fileOffset)
	}
	t.flowMu.Unlock()
	cache := torrstor.Global().CacheByHash([20]byte(t.Hash()))
	status := t.Status()
	for i := range out {
		s := &out[i]
		f := t.fileByID(s.FileIndex)
		if f == nil {
			continue
		}
		t.mu.Lock()
		bitrate, duration, probeFile := t.BitRate, t.DurationSeconds, t.ProbeFileID
		t.mu.Unlock()
		if probeFile != s.FileIndex {
			bitrate, duration = "", 0
		}
		e := flow.MediaEstimate(s.FileSize, duration, bitrate)
		s.EstimatedMediaBitrate = e.BytesPerSecond * 8
		if e.BytesPerSecond > 0 {
			s.PlaybackOffsetSeconds = float64(s.PlaybackOffsetBytes) / e.BytesPerSecond
		}
		s.BitrateEstimateSource, s.BitrateEstimateConfidence = e.Source, e.Confidence
		s.DownloadRate, s.UploadRate, s.ConnectedPeers = status.DownloadSpeed, status.UploadSpeed, status.ActivePeers
		s.SustainabilityRatio = flow.Sustainability(status.DownloadSpeed, e)
		if cache == nil || cache.PieceLength <= 0 {
			continue
		}
		s.CacheUsed = cache.Filled()
		s.CounterSnapshot = cache.FlowCounters()
		if sets := settings.BTsets(); sets != nil {
			s.CacheSize = sets.CacheSize
		}
		start := offsets[i] + s.PlaybackOffsetBytes
		end := offsets[i] + s.FileSize
		if start < offsets[i] || start >= end {
			continue
		}
		for pos := start; pos < end; {
			piece := int(pos / cache.PieceLength)
			if !cache.Have(piece) {
				break
			}
			next := (int64(piece) + 1) * cache.PieceLength
			if next > end {
				next = end
			}
			s.BufferAheadBytes += next - pos
			pos = next
		}
		s.BufferAheadSeconds = flow.BufferSeconds(s.BufferAheadBytes, e)
	}
	return out
}
