package torr

import (
	"encoding/hex"
	"errors"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"server/flow"
	"server/log"
	"server/lt"
	"server/settings"
	"server/torr/state"
	storageState "server/torr/storage/state"
	"server/torr/storage/torrstor"
	"server/torrshash"
	utils2 "server/utils"
)

// Torrent is the public engine-agnostic wrapper. Composes a libtorrent
// handle with the metadata (title/poster/category) the rest of the
// project decorates torrents with.
type Torrent struct {
	Title    string
	Category string
	Poster   string
	Data     string
	*TorrentSpec

	Stat      state.TorrentStat
	Timestamp int64
	Size      int64

	bt                        *BTServer
	lh                        atomic.Pointer[lt.Torrent]   // nil for DB-only or closed torrents
	filesSnapshot             atomic.Pointer[torrentFiles] // immutable after metadata arrives
	filesMu                   sync.Mutex
	nextEpisodeMu             sync.Mutex
	nextEpisodeCurrent        int
	nextEpisodeManual         int
	nextEpisodeTarget         int
	nextEpisodeAutomatic      bool
	nextEpisodeSuppressed     bool
	nextEpisodeSelectionKnown bool

	mu       sync.Mutex
	sourceMu sync.Mutex

	// runtime metrics, updated by watch()
	lastTimeSpeed       time.Time
	DownloadSpeed       float64
	UploadSpeed         float64
	BytesReadUsefulData int64
	BytesWrittenData    int64
	libraryNative       lt.Status // watch snapshot; guarded by mu

	// counters driven by the alert pump (atomic-friendly under mu)
	piecesDirtiedGood int64
	piecesDirtiedBad  int64

	PreloadSize    int64
	PreloadedBytes int64

	// play-gate state. The fill itself runs detached (context.Background) so an
	// impatient external player disconnecting mid-buffer can't abort it; the gate
	// tracks which file is filling and a channel that closes when it finishes, so
	// a reconnect for the SAME file WAITS for that same fill instead of skipping
	// the gate and streaming a half-buffer. preloadGateLast keeps the post-fill
	// debounce that suppresses the per-file preload storm a playlist client
	// triggers when it walks every &play entry of a multi-file torrent.
	preloadGateMu    sync.Mutex
	preloadGateIndex int
	preloadGateDone  chan struct{}
	preloadGateLast  time.Time

	// playStarted records that this torrent has begun playback once (on playStartIndex),
	// so the start-preload fires only at torrent START, not on every series switch: a
	// multi-file torrent is treated as a single unit. A play for a DIFFERENT file once
	// started streams directly (the reader's window buffers it), matching upstream
	// TorrServer, which preloads only on the explicit &preload — never on a play. Guarded
	// by preloadGateMu; reset implicitly when the torrent is dropped (fresh struct).
	playStarted    bool
	playStartIndex int

	DurationSeconds        float64
	BitRate                string
	ProbeFileID            int
	flowProbeFinishedIndex int
	flowStartupStarted     time.Time
	flowAddedAt            time.Time
	diagnosticID           uint64
	timeline               flow.Timeline
	preloadWorkMu          sync.Mutex
	preloadWork            *preloadOperation
	flowStartup            FlowStartupStatus

	flowMu           sync.Mutex
	flowSessions     map[string]*flowSession
	sparse           lt.SparseSnapshot // immutable cached aggregate, guarded by flowMu
	sparseRecordedAt int64
	peerHintsSavedAt time.Time
	requestTimeouts  atomic.Uint64
	requestsDropped  atomic.Uint64
	trackerMu        sync.Mutex
	trackers         map[string]FlowTrackerDiagnostic

	expiredTime     time.Time
	preparationHold atomic.Bool
	warmIdleSince   time.Time

	gotInfoCh   chan struct{}
	gotInfoOnce sync.Once
	posterOnce  sync.Once

	closeCh   chan struct{}
	closeOnce sync.Once

	watcher *time.Ticker
}

// NewTorrent installs a new torrent in the session.
func NewTorrent(spec *TorrentSpec, bt *BTServer) (*Torrent, error) {
	if bt == nil || bt.Session() == nil {
		return nil, errors.New("torr.NewTorrent: BT client not connected")
	}
	if spec == nil {
		return nil, errors.New("torr.NewTorrent: nil spec")
	}
	// A live handle needs no disk rehash or metadata parse. Recheck again under
	// the registration lock below when a new handle is actually needed.
	if existing := bt.GetTorrent(spec.InfoHash); existing != nil {
		existing.mu.Lock()
		closed := existing.Stat == state.TorrentClosed
		existing.mu.Unlock()
		if !closed {
			return existing, nil
		}
	}

	// Never mutate the stored specification with public discovery additions.
	// Unknown magnets use only their supplied trackers until metadata identifies
	// the torrent as public. Private metadata retains its authorized tiers.
	trackerTiers := spec.Trackers
	var metadata *lt.ParsedTorrent
	if len(spec.InfoBytes) > 0 {
		pt, parseErr := lt.ParseTorrentBytes(spec.InfoBytes)
		if parseErr != nil {
			return nil, parseErr
		}
		metadata = pt
		if pt.Private && len(pt.TrackerTiers) > 0 {
			trackerTiers = pt.TrackerTiers
		}
		if !pt.Private {
			trackerTiers = publicTrackerTiers(spec.Trackers)
		}
	}

	// If metadata is known at add time (InfoBytes present), scan the
	// per-piece cache dir for resume bits. Without metadata we don't
	// know the piece geometry yet; libtorrent will start downloading
	// and we'll discover have-state on subsequent restarts.
	var (
		havePieces []byte
		pieceCount int
	)
	if metadata != nil && metadata.HasMetadata && metadata.NumPieces > 0 {
		pieceCount = metadata.NumPieces
		if raw, err := hex.DecodeString(metadata.PieceHashes); err == nil && len(raw)/20 == pieceCount && len(raw)%20 == 0 {
			hashes := make([][20]byte, pieceCount)
			for i := range hashes {
				copy(hashes[i][:], raw[i*20:(i+1)*20])
			}
			if plan, ok := torrstor.Global().PreparationPlan(spec.InfoHash); ok {
				havePieces = flow.VerifyPieceFiles(filepath.Join(plan.Root, spec.InfoHash.HexString()), metadata.PieceLength, metadata.TotalSize, hashes)
			} else {
				havePieces = torrstor.ScanHavePieces(spec.InfoHash, pieceCount, metadata.PieceLength, metadata.TotalSize, hashes)
			}
		}
	}

	// Hold the registry lock across check + session add + register. The
	// check-then-register used to be two separate critical sections, so two
	// concurrent adds of the same hash (e.g. an HTTP handler re-adding an
	// idle-dropped torrent racing GetTorrent's async promotion) both passed
	// the check and produced two Go Torrents over one libtorrent torrent.
	// The loser was orphaned — alerts route by hash to the registered
	// instance only — so its GotInfo timed out 90s later and Close()d the
	// shared lt torrent out from under an active stream.
	bt.mu.Lock()
	if bt.session == nil {
		bt.mu.Unlock()
		return nil, errors.New("torr.NewTorrent: BT client not connected")
	}
	if existing := bt.torrents[spec.InfoHash]; existing != nil {
		existing.mu.Lock()
		closed := existing.Stat == state.TorrentClosed
		existing.mu.Unlock()
		if !closed {
			bt.mu.Unlock()
			return existing, nil
		}
		// A closed instance left in the registry (e.g. a magnet whose metadata
		// fetch timed out: GotInfo's failure path Close()s without deregistering,
		// and expired() skips TorrentClosed so expireWatch never reaps it) must
		// not shadow the hash forever — every re-add would get the zombie back
		// and insta-fail. Its lt torrent is already removed; replace the entry.
		delete(bt.torrents, spec.InfoHash)
	}

	if metadata != nil {
		torrstor.Global().SetVerifiedResume(spec.InfoHash, havePieces, metadata.TotalSize)
		if len(metadata.WebSeeds) > 0 {
			torrstor.Global().RequireVerifiedReads(spec.InfoHash)
		}
	}
	for _, seed := range spec.WebSeeds {
		if !seed.Disabled {
			torrstor.Global().RequireVerifiedReads(spec.InfoHash)
			break
		}
	}
	lh, err := bt.session.AddTorrent(lt.AddTorrentParams{
		Link:         magnetFromSpec(spec),
		InfoBytes:    spec.InfoBytes,
		TrackerTiers: trackerTiers,
		SavePath:     legacySavePath(spec.InfoHash),
		Paused:       flowPaused.Load(),
		HavePieces:   havePieces,
		PieceCount:   pieceCount,
	})
	if err != nil {
		torrstor.Global().ClearVerifiedResume(spec.InfoHash)
		bt.mu.Unlock()
		return nil, err
	}

	timeout := torrentExpireTimeout()
	t := &Torrent{
		TorrentSpec:   spec,
		bt:            bt,
		Stat:          state.TorrentAdded,
		Timestamp:     time.Now().Unix(),
		lastTimeSpeed: time.Now(),
		flowAddedAt:   time.Now(), diagnosticID: diagnosticSequence.Add(1),
		flowStartup: FlowStartupStatus{State: "METADATA", WaitReason: "METADATA", MetadataReadyMs: -1, FirstDHTPeerMs: -1, FirstPeerMs: -1, FirstUsefulBlockMs: -1},
		gotInfoCh:   make(chan struct{}),
		closeCh:     make(chan struct{}),
	}
	t.lh.Store(lh)
	bt.torrents[spec.InfoHash] = t
	bt.mu.Unlock()

	t.AddExpiredTime(timeout)

	// BTsets.ConnectionsLimit is a per-torrent peer cap (anacrolix legacy);
	// the session-wide connections_limit is set separately in
	// buildSessionConfig. See lt_shim.h on lt_torrent_set_max_connections.
	if lh != nil && settings.BTsets() != nil && settings.BTsets().ConnectionsLimit > 0 {
		_ = lh.SetMaxConnections(settings.BTsets().ConnectionsLimit)
	}

	// If the .torrent payload was provided up-front, metadata is already
	// known — signal immediately. Otherwise this is a magnet and we must pull
	// the info-dict from peers before anything (file list, playlist, stream)
	// can proceed; kick an immediate DHT + tracker announce so peer discovery
	// for the metadata starts aggressively right away instead of waiting for a
	// Reader to attach (which only happens at playback start, long after the
	// playlist is requested). This is the difference between a magnet whose
	// playlist appears in a couple of seconds and one that times out.
	if len(spec.InfoBytes) > 0 {
		t.signalGotInfo()
		for _, seed := range spec.WebSeeds {
			if _, err := flow.ValidateWebSeed(seed.URL, seed.AllowLocal); err == nil {
				_ = lh.SetURLSeed(seed.URL, seed.Disabled, seed.AllowLocal)
			}
		}
	} else if lh != nil {
		go func() {
			_ = lh.ForceReannounce()
			if settings.BTsets() == nil || !settings.BTsets().DisableDHT {
				_ = lh.ForceDhtAnnounce()
			}
		}()
	}

	go t.watch()
	if metadata != nil && !metadata.Private && settings.CurrentFlow().PeerResumeHints {
		go t.restorePeerHints(lh)
	}
	return t, nil
}

func magnetFromSpec(spec *TorrentSpec) string {
	if len(spec.InfoBytes) > 0 {
		return ""
	}
	return "magnet:?xt=urn:btih:" + spec.InfoHash.HexString()
}

func torrentExpireTimeout() time.Duration {
	if f := settings.CurrentFlow(); f.Enabled {
		return time.Duration(f.WarmSessionTimeoutSec) * time.Second
	}
	t := time.Second * time.Duration(settings.BTsets().TorrentDisconnectTimeout)
	if t > time.Minute {
		t = time.Minute
	}
	return t
}

func legacySavePath(h Hash) string {
	if settings.BTsets() != nil && settings.BTsets().UseDisk && settings.BTsets().TorrentsSavePath != "" {
		return settings.BTsets().TorrentsSavePath + "/" + h.HexString()
	}
	return ""
}

// ----- metadata ready signalling -----

func (t *Torrent) signalGotInfo() {
	// Only fire once metadata is actually parsed. The add_torrent alert (and a
	// magnet's early alerts) arrive BEFORE the info-dict is known; consuming the
	// once here would burn it on a no-op SetAllPiecesPriority and then make the
	// real metadata_received alert a no-op — leaving the torrent at libtorrent's
	// default piece priority 1, which downloads the WHOLE torrent and thrashes
	// the bounded cache. So bail until metadata is ready and wait for a later
	// alert (metadata_received / torrent_finished) or WaitInfo's fast path.
	//
	// Snapshot the handle once: Close() atomically detaches it, and
	// an instance that loses a concurrent add race (remove → immediate re-add,
	// e.g. /gst/remove followed by the next playlist request) is Closed while
	// its NewTorrent goroutine is still in here — the re-read of t.lh between
	// the nil check and the call segfaulted on the nil receiver. A call on a
	// CLOSED handle is harmless (the C slot lookup returns an error).
	lh := t.LTHandle()
	if lh == nil {
		return
	}
	if have, _ := lh.HaveMetadata(); !have {
		return
	}
	t.gotInfoOnce.Do(func() {
		t.mu.Lock()
		t.flowStartup.MetadataReadyMs = time.Since(t.flowAddedAt).Milliseconds()
		t.flowStartup.WaitReason = ""
		t.flowStartup.State = "IDLE"
		t.mu.Unlock()
		t.historyEvent(flow.HistoryEvent{Type: "metadata", ElapsedMs: time.Since(t.flowAddedAt).Milliseconds()})
		// Switch to lazy/streaming mode: download nothing until a Reader's
		// window or Preload bumps the specific pieces it needs.
		_ = lh.SetAllPiecesPriority(0)
		if st, err := lh.Status(); err == nil && !st.Private && len(t.InfoBytes) == 0 {
			_ = lh.ReplaceTrackers(publicTrackerTiers(t.Trackers))
		}
		// Backfill the spec with the just-received info-dict: a magnet-added
		// torrent's spec has no InfoBytes, so without this every DB save stores
		// the bare magnet and every server restart re-fetches metadata from the
		// swarm (slow playlists / slow first play after restart).
		if t.TorrentSpec != nil && len(t.TorrentSpec.InfoBytes) == 0 {
			if mb, err := lh.Metadata(); err == nil && len(mb) > 0 {
				t.mu.Lock()
				t.TorrentSpec.InfoBytes = mb
				t.mu.Unlock()
			}
		}
		if t.gotInfoCh != nil {
			close(t.gotInfoCh)
		}
	})
}

// WaitInfo blocks until metadata is received or the torrent is closed or
// the configured timeout elapses. Returns true on success.
func (t *Torrent) WaitInfo() bool {
	if t == nil {
		return false
	}
	// The captured native identity stays safe after concurrent removal.
	lh := t.LTHandle()
	if lh == nil {
		return false
	}
	// Fast path: already have metadata (e.g. info bytes were passed in).
	if have, _ := lh.HaveMetadata(); have {
		t.signalGotInfo()
		return true
	}
	deadline := time.Minute + time.Second*time.Duration(settings.BTsets().TorrentDisconnectTimeout)
	select {
	case <-t.gotInfoCh:
		return true
	case <-t.closeCh:
		return false
	case <-time.After(deadline):
		return false
	}
}

// GotInfo wraps WaitInfo with state transitions matching the legacy API.
func (t *Torrent) GotInfo() bool {
	if t == nil {
		return false
	}
	t.mu.Lock()
	if t.Stat == state.TorrentClosed {
		t.mu.Unlock()
		return false
	}
	if t.Stat == state.TorrentPreload {
		t.mu.Unlock()
		return true
	}
	t.Stat = state.TorrentGettingInfo
	t.mu.Unlock()
	if t.WaitInfo() {
		t.mu.Lock()
		if t.Stat == state.TorrentClosed {
			t.mu.Unlock()
			return false
		}
		t.Stat = state.TorrentWorking
		t.mu.Unlock()
		t.AddExpiredTime(torrentExpireTimeout())
		// Metadata (and so the release name) is now known — backfill a TMDB
		// poster for torrents that came in without one (bare magnets, tgbot,
		// autoload). No-op unless a TMDB API key is configured.
		go t.posterOnce.Do(t.fetchPosterIfMissing)
		return true
	}
	// Deregister before closing: expired() skips TorrentClosed, so a closed
	// instance left in the registry is never reaped and shadows its hash —
	// every later add of the same magnet returns the zombie and fails
	// instantly instead of retrying the metadata fetch.
	if t.bt != nil {
		t.bt.dropInstance(t)
	} else {
		t.Close()
	}
	return false
}

// AddExpiredTime pushes the auto-drop deadline forward.
func (t *Torrent) AddExpiredTime(d time.Duration) {
	newDeadline := time.Now().Add(d)
	t.mu.Lock()
	if t.expiredTime.Before(newDeadline) {
		t.expiredTime = newDeadline
	}
	t.mu.Unlock()
}

// expired reports whether the torrent has passed its auto-drop deadline and is
// safe to drop from the session: it must not be mid-handshake (getting metadata
// or preloading) or already closed, and must have no active reader. Playback,
// status polls and preload all push expiredTime forward, so an in-use torrent
// is never seen as expired. Used by BTServer.expireWatch.
func (t *Torrent) expired(now time.Time) bool {
	if t.preparationHold.Load() {
		return false
	}
	t.mu.Lock()
	stat := t.Stat
	deadline := t.expiredTime
	warmIdleSince := t.warmIdleSince
	t.mu.Unlock()
	switch stat {
	case state.TorrentClosed, state.TorrentGettingInfo, state.TorrentPreload, state.TorrentInDB:
		return false
	}
	if c := torrstor.Global().CacheByHash([20]byte(t.Hash())); c != nil && c.ActiveReaders() > 0 {
		return false
	}
	if settings.CurrentFlow().Enabled && !warmIdleSince.IsZero() {
		return !now.Before(warmIdleSince.Add(torrentExpireTimeout()))
	}
	if deadline.IsZero() || now.Before(deadline) {
		return false
	}
	return true
}

// ----- watch / progress -----

func (t *Torrent) watch() {
	t.watcher = time.NewTicker(time.Second)
	defer t.watcher.Stop()
	for {
		select {
		case <-t.closeCh:
			return
		case <-t.watcher.C:
			t.progressTick()
		}
	}
}

func (t *Torrent) progressTick() {
	handle := t.LTHandle()
	if handle == nil {
		return
	}
	st, err := handle.Status()
	if err != nil {
		return
	}
	t.mu.Lock()
	now := time.Now()
	dt := now.Sub(t.lastTimeSpeed).Seconds()
	if dt > 0 {
		dlDelta := st.TotalPayloadDownload - t.BytesReadUsefulData
		upDelta := st.TotalPayloadUpload - t.BytesWrittenData
		t.DownloadSpeed = float64(dlDelta) / dt
		t.UploadSpeed = float64(upDelta) / dt
	}
	t.BytesReadUsefulData = st.TotalPayloadDownload
	t.BytesWrittenData = st.TotalPayloadUpload
	t.lastTimeSpeed = now
	t.libraryNative = *st
	rate := t.DownloadSpeed
	t.mu.Unlock()
	if cache := torrstor.Global().CacheByHash([20]byte(t.Hash())); cache != nil {
		cache.SetFlowDownloadRate(rate, FlowIsPaused())
		t.tickNextEpisode(cache)
	}
	count := 0
	if st.PieceLength > 0 && st.TotalSize > 0 {
		count = int((st.TotalSize-1)/st.PieceLength + 1)
	}
	t.sampleSparse(handle, st.HasMetadata, st.PieceLength, count)
}

// ----- shutdown -----

// Close detaches the published handle, marks the torrent closed and removes
// its native handle if this instance still owns the registry entry.
func (t *Torrent) Close() bool {
	if t == nil {
		return false
	}
	t.mu.Lock()
	if t.Stat == state.TorrentClosed {
		t.mu.Unlock()
		t.markClosed()
		return true
	}
	t.Stat = state.TorrentClosed
	t.mu.Unlock()
	t.markClosed()
	hash := t.Hash()
	handle := t.lh.Swap(nil)
	if handle != nil && t.bt != nil {
		// Only remove the libtorrent torrent if no OTHER live instance owns
		// it. A duplicate Torrent that lost an add race (or any stale copy)
		// shares the same underlying lt torrent with the registered one;
		// removing it here would kill an active stream. The registry entry
		// is either us or already deleted (RemoveTorrent deletes before
		// closing) — both mean we own the removal.
		if cur := t.bt.GetTorrent(hash); cur == nil || cur == t {
			_ = handle.Remove(false)
		}
	}
	return true
}

func (t *Torrent) markClosed() {
	if cache := torrstor.Global().CacheByHash([20]byte(t.Hash())); cache != nil {
		cache.ClearNextEpisodeWarmup()
	}
	t.closeOnce.Do(func() {
		if t.closeCh != nil {
			close(t.closeCh)
		}
	})
	t.stopPreload()
}

// ----- accessors -----

// Hash returns the v1 info hash. Falls back to the spec if libtorrent
// handle is no longer valid.
func (t *Torrent) Hash() Hash {
	if t == nil {
		return Hash{}
	}
	if t.TorrentSpec != nil && !t.TorrentSpec.InfoHash.IsZero() {
		return t.TorrentSpec.InfoHash
	}
	if handle := t.LTHandle(); handle != nil {
		return NewHashFromHex(handle.InfoHash())
	}
	return Hash{}
}

// Name returns the torrent's display name (from metadata if available,
// otherwise from the spec's display_name / file name).
func (t *Torrent) Name() string {
	if handle := t.LTHandle(); handle != nil {
		if name := handle.DisplayName(); name != "" {
			return name
		}
	}
	if t.TorrentSpec != nil {
		return t.TorrentSpec.DisplayName
	}
	return ""
}

// Length returns the total payload size in bytes (0 before metadata).
func (t *Torrent) Length() int64 {
	handle := t.LTHandle()
	if handle == nil {
		return 0
	}
	return handle.TotalSize()
}

// Files returns the file list once metadata is known.
func (t *Torrent) Files() []*File {
	snapshot := t.fileSnapshot()
	if snapshot == nil {
		return nil
	}
	out := make([]*File, 0, len(snapshot.sorted))
	for _, file := range snapshot.sorted {
		copy := *file
		out = append(out, &copy)
	}
	return out
}

type torrentFiles struct {
	sorted []*File
	byID   map[int]*File
}

func (t *Torrent) fileSnapshot() *torrentFiles {
	if t == nil || t.LTHandle() == nil {
		return nil
	}
	if snapshot := t.filesSnapshot.Load(); snapshot != nil {
		return snapshot
	}
	t.filesMu.Lock()
	defer t.filesMu.Unlock()
	if snapshot := t.filesSnapshot.Load(); snapshot != nil {
		return snapshot
	}
	handle := t.LTHandle()
	if handle == nil {
		return nil
	}
	raw, err := handle.Files()
	if err != nil || len(raw) == 0 {
		return nil
	}
	out := make([]*File, 0, len(raw))
	for i := range raw {
		f := raw[i]
		out = append(out, &File{
			Index:  f.Index,
			Path:   f.Path,
			Length: f.Size,
			Offset: f.Offset,
		})
	}
	sort.Slice(out, func(i, j int) bool { return utils2.CompareStrings(out[i].Path, out[j].Path) })
	snapshot := &torrentFiles{sorted: out, byID: make(map[int]*File, len(out))}
	for _, f := range out {
		snapshot.byID[f.Index+1] = f
	}
	t.filesSnapshot.Store(snapshot)
	return snapshot
}

// LTHandle returns the underlying libtorrent handle for callers that
// need to set piece priorities / deadlines. May be nil for DB-only
// or closed torrents. Captured handles remain safe to call after removal.
func (t *Torrent) LTHandle() *lt.Torrent {
	if t == nil {
		return nil
	}
	return t.lh.Load()
}

// Status builds a state.TorrentStatus snapshot consumed by the web API.
func (t *Torrent) Status() *state.TorrentStatus {
	t.mu.Lock()
	defer t.mu.Unlock()

	st := new(state.TorrentStatus)
	st.Stat = t.Stat
	st.StatString = t.Stat.String()
	st.Title = t.Title
	st.Category = t.Category
	st.Poster = t.Poster
	st.Data = t.Data
	st.Timestamp = t.Timestamp
	st.TorrentSize = t.Size
	st.BitRate = t.BitRate
	st.DurationSeconds = t.DurationSeconds
	if t.TorrentSpec != nil {
		st.Hash = t.TorrentSpec.InfoHash.HexString()
	}

	handle := t.LTHandle()
	if handle == nil {
		// DB-resident torrent (not in the session): recover the file list from
		// the record's cached Data so playlists and the web file tree work
		// without waking the torrent (= without a swarm metadata fetch).
		st.FileStats = fileStatsFromData(t.Data)
		return st
	}
	lst, err := handle.Status()
	if err != nil {
		return st
	}
	st.Name = lst.Name
	if st.Hash == "" {
		st.Hash = lst.InfoHash
	}
	st.LoadedSize = lst.TotalDone
	st.DownloadSpeed = t.DownloadSpeed
	st.UploadSpeed = t.UploadSpeed
	st.TotalPeers = lst.ListPeers
	st.PendingPeers = lst.ConnectCandidates
	st.ActivePeers = lst.NumPeers
	st.ConnectedSeeders = lst.NumSeeds
	st.HalfOpenPeers = 0
	st.BytesWritten = lst.TotalUpload
	st.BytesWrittenData = lst.TotalPayloadUpload
	st.BytesRead = lst.TotalDownload
	st.BytesReadData = lst.TotalPayloadDownload
	st.BytesReadUsefulData = lst.TotalPayloadDownload
	st.PreloadedBytes = t.PreloadedBytes
	st.PreloadSize = t.PreloadSize
	// Expose the existing playback-reader count so the library can highlight
	// active torrents without polling detailed Flow snapshots for every card.
	if cache := torrstor.Global().CacheByHash([20]byte(t.Hash())); cache != nil {
		st.ActiveReaders = cache.StreamingReaders()
	}
	st.WarmIdle = st.ActiveReaders == 0 && settings.CurrentFlow().Enabled &&
		!t.warmIdleSince.IsZero() && time.Since(t.warmIdleSince) < torrentExpireTimeout()

	// libtorrent doesn't surface chunk counters directly via
	// torrent_status; approximate by dividing payload bytes by the
	// 16 KiB BitTorrent block size. Good enough for /cache + UI.
	const chunkSize = int64(16 * 1024)
	st.ChunksRead = lst.TotalPayloadDownload / chunkSize
	st.ChunksReadUseful = st.ChunksRead
	st.ChunksReadWasted = (lst.TotalDownload - lst.TotalPayloadDownload) / chunkSize
	st.ChunksWritten = lst.TotalPayloadUpload / chunkSize
	st.PiecesDirtiedGood = t.piecesDirtiedGood
	st.PiecesDirtiedBad = t.piecesDirtiedBad

	if !lst.HasMetadata {
		// Metadata still in flight — fall back to the DB-cached file list (if
		// this torrent was ever saved) so the UI isn't blank meanwhile.
		st.FileStats = fileStatsFromData(t.Data)
	} else {
		st.TorrentSize = lst.TotalSize
		st.FileStats = nil
		if files := t.Files(); len(files) > 0 {
			for _, f := range files {
				st.FileStats = append(st.FileStats, &state.TorrentFileStat{
					Id:     f.Index + 1, // legacy: 0 means undefined in the web UI
					Path:   f.Path,
					Length: f.Length,
				})
			}
			th := torrshash.New(st.Hash)
			th.AddField(torrshash.TagTitle, st.Title)
			th.AddField(torrshash.TagPoster, st.Poster)
			th.AddField(torrshash.TagCategory, st.Category)
			th.AddField(torrshash.TagSize, strconv.FormatInt(st.TorrentSize, 10))
			if t.TorrentSpec != nil && len(t.TorrentSpec.Trackers) > 0 {
				for _, tr := range t.TorrentSpec.Trackers[0] {
					th.AddField(torrshash.TagTracker, tr)
				}
			}
			if tok, err := torrshash.Pack(th); err == nil {
				st.TorrsHash = tok
			}
		}
	}
	return st
}

// ----- streaming / reader -----

// NewReader hands out an io.ReadSeekCloser over the requested file,
// backed by the Cache registered for this torrent. Returns nil if the
// cache hasn't been opened yet (metadata still in flight) or file is
// nil.
func (t *Torrent) NewReader(file *File) Reader {
	return t.NewReaderGroup(file, "")
}

// NewReaderGroup is NewReader with an explicit device/session key (the client IP)
// so concurrent streams of the same torrent from different devices each get their
// own isolated sliding window in the cache. The streaming HTTP path passes the
// caller's IP; internal consumers (DLNA index, tgbot, FUSE) use NewReader and
// share the default group.
func (t *Torrent) NewReaderGroup(file *File, group string) Reader {
	handle := t.LTHandle()
	if handle == nil || file == nil {
		return nil
	}
	cache := torrstor.Global().CacheByHash([20]byte(t.Hash()))
	if cache == nil {
		log.TLogln("torr.NewReader: no cache for", t.Hash().HexString())
		return nil
	}
	return torrstor.NewReader(cache, handle, torrstor.FileInfo{
		Index:  file.Index,
		Path:   file.Path,
		Offset: file.Offset,
		Length: file.Length,
	}, group)
}

// CloseReader releases a previously handed-out reader and pushes the
// torrent's auto-drop deadline forward.
func (t *Torrent) CloseReader(r Reader) {
	if r != nil {
		_ = r.Close()
	}
	t.AddExpiredTime(torrentExpireTimeout())
}

// CacheState combines authoritative piece/reader accounting with torrent
// status, returning an empty cache map when its native storage is closed.
func (t *Torrent) CacheState() *storageState.CacheState {
	if t == nil {
		return &storageState.CacheState{Pieces: map[int]storageState.ItemState{}}
	}
	// Pull the real stats (Capacity, Filled, per-piece state) from the live
	// torrstor cache; fall back to an empty shell if it isn't open yet.
	var st *storageState.CacheState
	if c := torrstor.Global().CacheByHash([20]byte(t.Hash())); c != nil {
		st = c.State()
	} else {
		st = &storageState.CacheState{Pieces: map[int]storageState.ItemState{}}
	}
	st.Torrent = t.Status()
	if t.TorrentSpec != nil {
		st.Hash = t.TorrentSpec.InfoHash.HexString()
	}
	if handle := t.LTHandle(); handle != nil {
		st.PiecesCount = handle.NumPieces()
		st.PiecesLength = handle.PieceLength()
	}
	// Never hand the web UI a nil Pieces/Readers: useCreateCacheMap iterates both
	// without a null guard, so JSON null crashes the torrent info dialog.
	if st.Pieces == nil {
		st.Pieces = map[int]storageState.ItemState{}
	}
	if st.Readers == nil {
		st.Readers = []*storageState.ReaderState{}
	}
	return st
}
