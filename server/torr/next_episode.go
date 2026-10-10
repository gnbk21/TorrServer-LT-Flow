package torr

import (
	"errors"
	"server/flow"
	"server/settings"
	"server/torr/storage/torrstor"
	"server/utils"
)

type NextEpisodeStatus struct {
	torrstor.NextEpisodeStatus
	Enabled          bool `json:"enabled"`
	CurrentFileIndex int  `json:"current_file_index"`
}

func (t *Torrent) nextEpisodeFileChanged(fileID int) {
	t.nextEpisodeMu.Lock()
	defer t.nextEpisodeMu.Unlock()
	if t.nextEpisodeCurrent == fileID {
		return
	}
	t.nextEpisodeCurrent = fileID
	t.nextEpisodeManual, t.nextEpisodeTarget = 0, 0
	t.nextEpisodeSuppressed = false
	if cache := torrstor.Global().CacheByHash([20]byte(t.Hash())); cache != nil {
		cache.ClearNextEpisodeWarmup()
	}
}

func (t *Torrent) tickNextEpisode(cache *torrstor.Cache) {
	f := settings.CurrentFlow()
	t.nextEpisodeMu.Lock()
	defer t.nextEpisodeMu.Unlock()
	if !f.Enabled || !f.NextEpisodeWarmup || !f.MetricsEnabled || FlowIsPaused() || t.nextEpisodeSuppressed {
		cache.ClearNextEpisodeWarmup()
		return
	}
	current := t.nextEpisodeCurrent
	if current == 0 {
		cache.ClearNextEpisodeWarmup()
		return
	}
	eligible, seen, reason := true, false, "HEALTHY_BUFFER"
	for _, s := range t.FlowStatusWithTraces(false) {
		if s.State == "EXPIRED" {
			continue
		}
		if s.FileIndex != current && s.ActiveReaders > 0 {
			eligible = false
			reason = "MULTIPLE_FILES"
			continue
		}
		if s.FileIndex != current {
			continue
		}
		seen = true
		if s.Risk.Level != "HEALTHY" || s.Risk.Confidence == "unknown" || s.Delivery.AgeMs < 0 || s.Delivery.AgeMs > 5000 || s.BufferAheadSeconds < float64(max(30, f.TargetBufferSeconds)) {
			eligible = false
			reason = "PLAYBACK_RESERVE"
		}
	}
	if !seen {
		cache.ClearNextEpisodeWarmup()
		return
	}
	target := t.nextEpisodeManual
	automatic := target == 0
	if automatic {
		snapshot := t.fileSnapshot()
		if snapshot == nil {
			cache.ClearNextEpisodeWarmup()
			return
		}
		files := make([]flow.EpisodeFile, 0, len(snapshot.sorted))
		for _, file := range snapshot.sorted {
			files = append(files, flow.EpisodeFile{ID: file.Index + 1, Path: file.Path, Offset: file.Offset, Length: file.Length})
		}
		next, ok := flow.ConfidentNextEpisode(files, current)
		if !ok {
			cache.ClearNextEpisodeWarmup()
			t.nextEpisodeTarget = 0
			return
		}
		target = next.ID
	}
	file := t.fileByID(target)
	if file == nil {
		cache.ClearNextEpisodeWarmup()
		return
	}
	t.nextEpisodeTarget, t.nextEpisodeAutomatic = target, automatic
	cache.SetNextEpisodeWarmup(target, file.Offset, file.Length, automatic, eligible, reason)
}

func (t *Torrent) NextEpisodeWarmupStatus() NextEpisodeStatus {
	t.nextEpisodeMu.Lock()
	defer t.nextEpisodeMu.Unlock()
	f := settings.CurrentFlow()
	s := NextEpisodeStatus{Enabled: f.Enabled && f.NextEpisodeWarmup && f.MetricsEnabled, CurrentFileIndex: t.nextEpisodeCurrent}
	if cache := torrstor.Global().CacheByHash([20]byte(t.Hash())); cache != nil {
		s.NextEpisodeStatus = cache.NextEpisodeWarmupStatus()
	}
	if s.State == "" {
		s.State = "idle"
	}
	if s.State == "idle" && s.Enabled && s.CurrentFileIndex > 0 && !t.nextEpisodeSuppressed {
		s.Reason = "MANUAL_SELECTION_REQUIRED"
	}
	if t.nextEpisodeSuppressed {
		s.Reason = "CANCELLED"
	}
	return s
}

func (t *Torrent) SelectNextEpisodeWarmup(index int, action string) error {
	if t == nil || t.LTHandle() == nil {
		return errors.New("warmup requires a live torrent")
	}
	f := settings.CurrentFlow()
	if !f.Enabled || !f.NextEpisodeWarmup {
		return errors.New("next episode warmup is disabled")
	}
	t.nextEpisodeMu.Lock()
	defer t.nextEpisodeMu.Unlock()
	if t.nextEpisodeCurrent == 0 {
		return errors.New("warmup requires an observed playback file")
	}
	switch action {
	case "select":
		file := t.fileByID(index)
		if file == nil || index == t.nextEpisodeCurrent || utils.GetMimeType(file.Path) != "video/*" {
			return errors.New("invalid next episode file")
		}
		t.nextEpisodeManual = index
		t.nextEpisodeSuppressed = false
	case "auto":
		t.nextEpisodeManual = 0
		t.nextEpisodeSuppressed = false
	case "cancel":
		t.nextEpisodeManual = 0
		t.nextEpisodeSuppressed = true
	default:
		return errors.New("unknown warmup action")
	}
	t.nextEpisodeTarget = 0
	if cache := torrstor.Global().CacheByHash([20]byte(t.Hash())); cache != nil {
		cache.ClearNextEpisodeWarmup()
	}
	return nil
}
