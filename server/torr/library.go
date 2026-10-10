package torr

import (
	"server/settings"
	"server/torr/state"
	"server/torr/storage/torrstor"
	"time"
)

// LibrarySummary deliberately omits client Data, tracker-bearing links and file
// trees. Opening a file dialog still uses the legacy detail endpoint. This read
// neither wakes an idle torrent nor extends its lifetime or queries native peers.
func (t *Torrent) LibrarySummary() state.TorrentStatus {
	t.mu.Lock()
	row := state.TorrentStatus{Title: t.Title, Category: t.Category, Poster: t.Poster,
		Timestamp: t.Timestamp, Stat: t.Stat, StatString: t.Stat.String(),
		TorrentSize: t.Size, DownloadSpeed: t.DownloadSpeed, UploadSpeed: t.UploadSpeed,
		PreloadedBytes: t.PreloadedBytes, PreloadSize: t.PreloadSize}
	warmSince := t.warmIdleSince
	native := t.libraryNative
	row.Name = native.Name
	row.ActivePeers, row.TotalPeers, row.PendingPeers, row.ConnectedSeeders = native.NumPeers, native.ListPeers, native.ConnectCandidates, native.NumSeeds
	row.LoadedSize = native.TotalDone
	if native.TotalSize > 0 {
		row.TorrentSize = native.TotalSize
	}
	if t.TorrentSpec != nil {
		row.Hash = t.InfoHash.HexString()
	}
	t.mu.Unlock()
	if cache := torrstor.Global().CacheByHash([20]byte(t.Hash())); cache != nil {
		row.ActiveReaders = cache.StreamingReaders()
	}
	row.WarmIdle = row.ActiveReaders == 0 && settings.CurrentFlow().Enabled &&
		!warmSince.IsZero() && time.Since(warmSince) < torrentExpireTimeout()
	return row
}

// LibraryTorrents avoids the legacy sorting pass over mutable Torrent fields.
// Consumers sort their immutable summaries, after filtering the complete set.
func LibraryTorrents() []*Torrent {
	rows := ListTorrentsDB()
	if engine := helperEngine(); engine != nil {
		for hash, torrent := range engine.ListTorrents() {
			rows[hash] = torrent
		}
	}
	out := make([]*Torrent, 0, len(rows))
	for _, torrent := range rows {
		out = append(out, torrent)
	}
	return out
}

func ActiveTorrents() []*Torrent {
	out := []*Torrent{}
	if engine := helperEngine(); engine != nil {
		for _, torrent := range engine.ListTorrents() {
			out = append(out, torrent)
		}
	}
	return out
}

// Only selected playback files travel with the live projection, never the
// torrent's full tree. The legacy detail endpoint retains that complete tree.
func (t *Torrent) PlaybackFileSummary(index int) *state.TorrentFileStat {
	if file := t.fileByID(index); file != nil {
		return &state.TorrentFileStat{Id: index, Path: file.Path, Length: file.Length}
	}
	return nil
}
