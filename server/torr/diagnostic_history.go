package torr

import (
	"sort"
	"time"

	"server/flow"
	"server/settings"
	"server/torr/storage/torrstor"
)

func (t *Torrent) recordStartupProgress(progress flow.StartupHistory, force bool) {
	now := time.Now()
	sample := t.SparseStatus()
	progress.HeadSupply = startupPieceSupply(sample, progress.HeadBlockerPiece, now)
	progress.TailSupply = startupPieceSupply(sample, progress.TailBlockerPiece, now)
	t.mu.Lock()
	progress.WaitBudgetMs, progress.WaitExtended = t.flowStartup.WaitBudgetMs, t.flowStartup.WaitExtended
	t.flowStartup.Progress = &progress // immutable snapshot, also available in the UI
	file := t.flowStartup.FileIndex
	elapsed := now.Sub(t.flowStartupStarted).Milliseconds()
	record := force || now.Sub(t.startupRecordedAt) >= 2*time.Second
	if record {
		t.startupRecordedAt = now
	}
	t.mu.Unlock()
	if record && settings.CurrentFlow().MetricsEnabled {
		t.historyEvent(flow.HistoryEvent{Type: "startup_sample", File: file, ElapsedMs: elapsed, OperationMeasured: true, Startup: &progress})
	}
}

// The existing one-second torrent watcher owns sampling. Nothing is polled on
// the HTTP read path and opening the diagnostics drawer is not required.
func (t *Torrent) recordPlaybackHistory(now time.Time) {
	f := settings.CurrentFlow()
	if (!f.DiagnosticHistory && !f.IncidentFlightRecorder) || !f.MetricsEnabled || t.bt == nil {
		return
	}
	t.flowMu.Lock()
	if now.Sub(t.playbackRecordedAt) < 10*time.Second {
		t.flowMu.Unlock()
		return
	}
	t.playbackRecordedAt = now
	type identity struct {
		id   uint64
		seen time.Time
	}
	identities := make(map[string]identity)
	for _, s := range t.flowSessions {
		if s.Group != torrstor.ProbeReaderGroup && (s.ActiveReaders > 0 || now.Sub(s.lastSeen) < 30*time.Second) {
			identities[s.SessionID] = identity{s.diagnosticID, s.lastSeen}
		}
	}
	t.flowMu.Unlock()
	if len(identities) == 0 {
		return
	}
	sessions := t.FlowStatusWithTraces(false)
	// Keep the most recent demand first, with a deterministic, bounded sample.
	sort.Slice(sessions, func(i, j int) bool {
		a, b := identities[sessions[i].SessionID], identities[sessions[j].SessionID]
		if a.seen.Equal(b.seen) {
			return a.id < b.id
		}
		return a.seen.After(b.seen)
	})
	cache := torrstor.Global().CacheByHash(t.StorageHash())
	count := 0
	for _, s := range sessions {
		identity, ok := identities[s.SessionID]
		if !ok {
			continue
		}
		if !f.DiagnosticHistory {
			continue
		}
		p := flow.PlaybackHistory{Session: identity.id, State: s.State, DemandOffsetBytes: s.DemandOffsetBytes, DeliveredOffsetBytes: s.PlaybackOffsetBytes, BufferBytes: s.BufferAheadBytes, BufferSeconds: s.BufferAheadSeconds, DeliveryRate: s.Delivery.LongRate, DownloadRate: s.DownloadRate, ConsumptionRate: s.PlaybackConsumptionRate, SupplyQualified: s.SupplyQualified, WaitReason: s.WaitReason, ActiveReaders: s.ActiveReaders, RangeRequests: s.RangeRequestCount, RangeCancels: s.RangeCancelCount, RecentReadStalls: s.RecentServerReadStalls, RecentWaitP95Ms: s.RecentPieceWaitP95Ms, Suppliers: s.RequiredPieceSuppliers}
		if f := t.fileByID(s.FileIndex); f != nil && cache != nil && cache.PieceLength > 0 && s.DemandOffsetBytes < f.Length {
			p.RequiredPiece, p.RequiredPieceKnown = int((f.Offset+s.DemandOffsetBytes)/cache.PieceLength), true
		}
		t.historyEvent(flow.HistoryEvent{Type: "playback_sample", File: s.FileIndex, Playback: &p, Truncated: len(identities) > 8})
		count++
		if count == 8 {
			break
		}
	}
}
