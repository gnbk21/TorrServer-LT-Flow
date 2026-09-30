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
	TimeToFirstByteMs        int64  `json:"time_to_first_byte_ms"`
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
	BufferExhaustionSeconds   *float64         `json:"buffer_exhaustion_seconds,omitempty"`
	BufferWarning             bool             `json:"buffer_warning"`
	EstimatedMediaBitrate     float64          `json:"estimated_media_bitrate"`
	ObservedPlaybackRate      float64          `json:"observed_playback_rate"`
	ObservedConfidence        string           `json:"observed_confidence"`
	PlaybackConsumptionRate   float64          `json:"playback_consumption_rate"`
	TargetBufferSeconds       int              `json:"target_buffer_seconds"`
	ForwardWindowPieces       int              `json:"forward_window_pieces"`
	BitrateEstimateSource     string           `json:"bitrate_estimate_source"`
	BitrateEstimateConfidence string           `json:"bitrate_estimate_confidence"`
	DownloadRate              float64          `json:"download_rate"`
	RecentDownloadRate        float64          `json:"recent_download_rate"`
	DownloadRateSamples       int              `json:"download_rate_samples"`
	UploadRate                float64          `json:"upload_rate"`
	SustainabilityRatio       float64          `json:"sustainability_ratio"`
	CacheUsed                 int64            `json:"cache_used"`
	CacheSize                 int64            `json:"cache_size"`
	ConnectedPeers            int              `json:"connected_peers"`
	RangeRequestCount         uint64           `json:"range_request_count"`
	RangeCancelCount          uint64           `json:"range_cancel_count"`
	SeekCount                 uint64           `json:"seek_count"`
	SeekRecoveryMs            int64            `json:"seek_recovery_ms"`
	WarmReconnectCount        uint64           `json:"warm_reconnect_count"`
	WarmReconnectTTFBMs       int64            `json:"warm_reconnect_ttfb_ms"`
	LastClassification        string           `json:"last_classification"`
	LastTTFBMs                int64            `json:"last_ttfb_ms"`
	Traces                    []FlowRangeTrace `json:"traces,omitempty"`
}

type flowSession struct {
	FlowSessionStatus
	fileOffset      int64
	lastSeen        time.Time
	lastPlaybackSeq uint64
	lastSeekSeq     uint64
	lastWarmSeq     uint64
}

// flowStart records a logical playback session separately from a TCP request.
func (t *Torrent) flowStart(fileID int, file *File, group string, req *http.Request) (string, flow.RangeHint, uint64) {
	hint := flow.ParseRangeHint(req.Header.Get("Range"), file.Length)
	if t == nil {
		return "UNKNOWN", hint, 0
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
		return "UNKNOWN", hint, 0
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
	purpose := flow.Classify(req.Method, internal, hint, file.Length, s.PlaybackOffsetBytes, s.RangeRequestCount > 0)
	classification := purpose
	if !settings.CurrentFlow().RangeClassification {
		classification = "UNKNOWN"
	}
	if !internal && req.Method == http.MethodGet {
		if s.State == "WARM_IDLE" {
			s.WarmReconnectCount++
			s.lastWarmSeq = s.RangeRequestCount + 1
		}
		if purpose == "SEEK" {
			if cache := torrstor.Global().CacheByHash([20]byte(t.Hash())); cache != nil {
				cache.ResetFlowWindow(group, file.Index)
			}
			s.SeekCount++
			s.lastSeekSeq = s.RangeRequestCount + 1
			s.State = "SEEK_RECOVERY"
		} else {
			s.State = "PLAYING"
		}
		s.ActiveReaders++
	}
	s.RangeRequestCount++
	seq := s.RangeRequestCount
	if !internal && req.Method == http.MethodGet && purpose != "HEAD_PROBE" && purpose != "TAIL_INDEX" {
		s.lastPlaybackSeq = seq
	}
	s.LastClassification = classification
	s.lastSeen = time.Now()
	return classification, hint, seq
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

// flowProgress follows delivered bytes throughout an open HTTP response. This
// is the server delivery position, not the player's decoded presentation time.
func (t *Torrent) flowProgress(fileID int, group string, seq uint64, offset int64) {
	if t == nil || seq == 0 {
		return
	}
	t.flowMu.Lock()
	defer t.flowMu.Unlock()
	if s := t.flowSessions[fmt.Sprintf("%d/%s", fileID, group)]; s != nil && seq == s.lastPlaybackSeq {
		s.PlaybackOffsetBytes = max(int64(0), min(offset, s.FileSize))
		s.lastSeen = time.Now()
	}
}

func (t *Torrent) flowEnd(fileID int, group string, seq uint64, tr FlowRangeTrace) {
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
	if tr.BytesServed > 0 && (tr.Status == http.StatusOK || tr.Status == http.StatusPartialContent) {
		if seq == s.lastSeekSeq {
			s.SeekRecoveryMs = tr.TTFBMs
		}
		if seq == s.lastWarmSeq {
			s.WarmReconnectTTFBMs = tr.TTFBMs
		}
	}
	if group != torrstor.ProbeReaderGroup && tr.Method == http.MethodGet {
		if s.ActiveReaders > 0 {
			s.ActiveReaders--
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

func (t *Torrent) flowFirstByte(fileID int, group string, seq uint64, started time.Time, ttfb time.Duration) {
	if t == nil || group == torrstor.ProbeReaderGroup {
		return
	}
	t.mu.Lock()
	if t.flowStartup.FileIndex == fileID && !t.flowStartupStarted.IsZero() && t.flowStartup.TimeToFirstByteMs == 0 {
		t.flowStartup.TimeToFirstByteMs = started.Sub(t.flowStartupStarted).Milliseconds() + ttfb.Milliseconds()
	}
	t.mu.Unlock()
	t.flowMu.Lock()
	if s := t.flowSessions[fmt.Sprintf("%d/%s", fileID, group)]; s != nil {
		if seq == s.lastSeekSeq {
			s.SeekRecoveryMs = ttfb.Milliseconds()
		}
		if seq == s.lastWarmSeq {
			s.WarmReconnectTTFBMs = ttfb.Milliseconds()
		}
	}
	t.flowMu.Unlock()
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
			if key, ok := t.probeKey(s.FileIndex); ok {
				if cached, ok := mediaProbes.Get(key, time.Now()); ok {
					bitrate, duration = cached.BitRate, cached.Duration
				}
			}
		}
		e := flow.MediaEstimate(s.FileSize, duration, bitrate)
		s.EstimatedMediaBitrate = e.BytesPerSecond * 8
		if e.BytesPerSecond > 0 {
			s.PlaybackOffsetSeconds = float64(s.PlaybackOffsetBytes) / e.BytesPerSecond
		}
		s.BitrateEstimateSource, s.BitrateEstimateConfidence = e.Source, e.Confidence
		if cache != nil {
			w := cache.FlowWindow(s.Group)
			s.RecentDownloadRate, s.DownloadRateSamples = w.RecentDownloadRate, w.DownloadRateSamples
			s.ObservedPlaybackRate, s.ObservedConfidence = w.ObservedPlaybackRate, w.ObservedConfidence
			s.TargetBufferSeconds, s.ForwardWindowPieces = w.TargetBufferSeconds, w.ForwardWindowPieces
		}
		s.PlaybackConsumptionRate = e.BytesPerSecond
		if s.ObservedPlaybackRate > 0 {
			s.PlaybackConsumptionRate = s.ObservedPlaybackRate
		}
		s.DownloadRate, s.UploadRate, s.ConnectedPeers = status.DownloadSpeed, status.UploadSpeed, status.ActivePeers
		rate := s.DownloadRate
		if s.DownloadRateSamples > 0 {
			rate = s.RecentDownloadRate
		}
		s.SustainabilityRatio = flow.Sustainability(rate,
			flow.Estimate{BytesPerSecond: s.PlaybackConsumptionRate})
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
		s.BufferAheadBytes = cache.ContiguousAvailable(start, end)
		s.BufferAheadSeconds = flow.BufferSeconds(s.BufferAheadBytes,
			flow.Estimate{BytesPerSecond: s.PlaybackConsumptionRate})
		if s.ActiveReaders > 0 && s.PlaybackOffsetBytes > 0 && s.BufferAheadBytes < end-start {
			if seconds, known := flow.BufferExhaustionSeconds(s.BufferAheadBytes, s.PlaybackConsumptionRate, rate); known {
				s.BufferExhaustionSeconds = &seconds
				s.BufferWarning = seconds < 30
			}
		}
	}
	return out
}
