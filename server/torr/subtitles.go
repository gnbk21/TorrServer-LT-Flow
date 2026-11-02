package torr

import (
	"context"
	"errors"
	"io"
	"server/flow"
	"server/torr/storage/torrstor"
	"server/utils"
)

func (t *Torrent) SubtitleCandidates(videoID int) ([]flow.SubtitleCandidate, bool) {
	files := t.subtitleFiles()
	if utils.GetMimeType(t.SubtitleVideoPath(videoID)) != "video/*" {
		return []flow.SubtitleCandidate{}, false
	}
	return flow.MatchSubtitles(files, videoID)
}
func (t *Torrent) subtitleFiles() []flow.EpisodeFile {
	files := []flow.EpisodeFile{}
	if t == nil {
		return files
	}
	if snapshot := t.filesSnapshot.Load(); snapshot != nil {
		for _, f := range snapshot.sorted {
			files = append(files, flow.EpisodeFile{ID: f.Index + 1, Path: f.Path, Offset: f.Offset, Length: f.Length})
		}
		return files
	}
	t.mu.Lock()
	data := t.Data
	t.mu.Unlock()
	for _, f := range fileStatsFromData(data) {
		files = append(files, flow.EpisodeFile{ID: f.Id, Path: f.Path, Length: f.Length})
	}
	return files
}
func (t *Torrent) SubtitleVideoPath(index int) string {
	for _, file := range t.subtitleFiles() {
		if file.ID == index {
			return file.Path
		}
	}
	return ""
}
func (t *Torrent) SelectedSubtitle(videoID, track int) (flow.SubtitleCandidate, bool) {
	rows, _ := t.SubtitleCandidates(videoID)
	for _, row := range rows {
		if row.FileIndex == track {
			return row, true
		}
	}
	return flow.SubtitleCandidate{}, false
}
func (t *Torrent) SubtitleBytes(ctx context.Context, videoID, track int) ([]byte, flow.SubtitleCandidate, error) {
	candidate, found := t.SelectedSubtitle(videoID, track)
	if !found {
		return nil, candidate, errors.New("subtitle does not match the selected video")
	}
	if candidate.Size > flow.SubtitleInputLimit {
		return nil, candidate, flow.ErrSubtitle
	}
	if !t.beginStream() {
		return nil, candidate, torrstor.ErrAuxiliaryDeferred
	}
	defer t.endStream()
	if FlowIsPaused() {
		return nil, candidate, torrstor.ErrAuxiliaryDeferred
	}
	for _, session := range t.FlowStatusWithTraces(false) {
		if session.ActiveReaders > 0 && (session.Risk.Level != "HEALTHY" || session.Risk.Confidence == "unknown") {
			return nil, candidate, torrstor.ErrAuxiliaryDeferred
		}
	}
	cache := torrstor.Global().CacheByHash(t.StorageHash())
	if cache == nil {
		return nil, candidate, torrstor.ErrAuxiliaryDeferred
	}
	file := t.fileByID(track)
	if file == nil {
		return nil, candidate, flow.ErrSubtitle
	}
	reader, err := cache.NewAuxiliaryReader(ctx, torrstor.FileInfo{Index: file.Index, Path: file.Path, Offset: file.Offset, Length: file.Length})
	if err != nil {
		return nil, candidate, err
	}
	defer reader.Close()
	bytes, err := io.ReadAll(io.LimitReader(reader, flow.SubtitleInputLimit+1))
	// Priority/pressure cancellation is temporary unavailability, not malformed
	// subtitle content. Preserve the caller's own cancellation separately.
	if err != nil && ctx.Err() == nil && (errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)) {
		err = torrstor.ErrAuxiliaryDeferred
	}
	if err == nil && int64(len(bytes)) != candidate.Size {
		err = flow.ErrSubtitle
	}
	return bytes, candidate, err
}
