package torr

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"time"

	"server/flow"
	"server/lt"
	"server/settings"
	"server/torr/storage/torrstor"
)

const maxPreparationJobs = 16

type PreparationJob struct {
	ID              string `json:"id"`
	Hash            string `json:"hash"`
	FileIndex       int    `json:"file_index"`
	State           string `json:"state"`
	Length          int64  `json:"length"`
	VerifiedBytes   int64  `json:"verified_bytes"`
	ContiguousBytes int64  `json:"contiguous_bytes"`
	PlaybackReady   bool   `json:"playback_ready"`
	ErrorCode       string `json:"error_code,omitempty"`
}
type preparationRecord struct {
	PreparationJob
	Spec                   TorrentSpec
	Offset                 int64
	First, Last            int
	PieceLength, TotalSize int64
}
type PreparationStatus struct {
	Jobs          []PreparationJob `json:"jobs"`
	QuotaBytes    int64            `json:"quota_bytes"`
	ReservedBytes int64            `json:"reserved_bytes"`
	ErrorCode     string           `json:"error_code,omitempty"`
}
type preparationManager struct {
	mu                  sync.Mutex
	bt                  *BTServer
	jobs                map[string]*preparationRecord
	name, root, failure string
}

func newPreparationManager(bt *BTServer) *preparationManager {
	torrstor.Global().ResetPreparationPlans()
	root := filepath.Join(settings.Path, "flow-pieces")
	if s := settings.BTsets(); s != nil && s.UseDisk && s.TorrentsSavePath != "" {
		root = s.TorrentsSavePath
	}
	p := &preparationManager{bt: bt, jobs: make(map[string]*preparationRecord), name: filepath.Join(settings.Path, "flow-preparation.json"), root: root}
	if settings.ReadOnly {
		p.failure = "READ_ONLY"
		return p
	}
	var saved []*preparationRecord
	if err := flow.ReadPreparationState(p.name, &saved); err != nil && !os.IsNotExist(err) {
		p.failure = "STATE_READ"
		return p
	}
	if len(saved) > maxPreparationJobs {
		p.failure = "STATE_READ"
		return p
	}
	for _, j := range saved {
		if j == nil || j.PieceLength <= 0 || j.TotalSize <= 0 || j.FileIndex < 1 || j.Spec.InfoHash.IsZero() || j.ID != preparationID(j.Spec.InfoHash, j.FileIndex) || j.Hash != j.Spec.InfoHash.HexString() || j.Length <= 0 || j.Offset < 0 || j.Offset > j.TotalSize-j.Length || j.First != int(j.Offset/j.PieceLength) || j.Last != int((j.Offset+j.Length-1)/j.PieceLength) {
			p.failure = "STATE_READ"
			p.jobs = make(map[string]*preparationRecord)
			return p
		}
		metadata, e := lt.ParseTorrentBytes(j.Spec.InfoBytes)
		if e != nil || metadata.InfoHash != j.Hash || metadata.TotalSize != j.TotalSize || metadata.PieceLength != j.PieceLength || len(metadata.PieceHashes) != metadata.NumPieces*40 {
			p.failure = "STATE_READ"
			p.jobs = make(map[string]*preparationRecord)
			return p
		}
		switch j.State {
		case "downloading", "ready", "paused", "cancelled", "error", "cleaning":
		default:
			p.failure = "STATE_READ"
			return p
		}
		j.VerifiedBytes = 0
		j.ContiguousBytes = 0
		j.PlaybackReady = false
		if j.State == "ready" {
			j.State = "downloading"
		} // readiness must be reverified
		p.jobs[j.ID] = j
	}
	if _, err := p.reserved(); err != nil {
		p.failure = "STATE_READ"
		p.jobs = make(map[string]*preparationRecord)
		return p
	}
	for _, j := range p.jobs {
		if reserved, _ := p.reserved(); reserved > int64(settings.CurrentFlow().PreparationQuotaMB)<<20 && j.State == "downloading" {
			j.State = "paused"
			j.ErrorCode = "QUOTA"
		}
		_ = p.configure(j.Spec.InfoHash)
	}
	return p
}

func preparationID(hash Hash, index int) string { return hash.HexString() + ":" + strconv.Itoa(index) }
func (p *preparationManager) records() []*preparationRecord {
	out := make([]*preparationRecord, 0, len(p.jobs))
	for _, j := range p.jobs {
		out = append(out, j)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
func (p *preparationManager) save() error {
	if settings.ReadOnly {
		return errors.New("read-only state")
	}
	return flow.WritePreparationState(p.name, p.records())
}
func (p *preparationManager) reserved() (int64, error) {
	var rs []flow.PieceReservation
	for _, j := range p.jobs {
		rs = append(rs, flow.PieceReservation{Hash: j.Hash, First: j.First, Last: j.Last, PieceLength: j.PieceLength, TotalSize: j.TotalSize})
	}
	return flow.PreparationReserved(rs)
}
func (p *preparationManager) configure(hash Hash) error {
	ranges := make(map[string]torrstor.PreparationRange)
	for id, j := range p.jobs {
		if j.Spec.InfoHash == hash {
			ranges[id] = torrstor.PreparationRange{First: j.First, Last: j.Last, Active: j.State == "downloading"}
		}
	}
	return torrstor.Global().ConfigurePreparation(hash, torrstor.PreparationStorage{Root: p.root, Ranges: ranges})
}
func PreparationSnapshot() PreparationStatus {
	if bts == nil || bts.preparation == nil {
		return PreparationStatus{Jobs: []PreparationJob{}, ErrorCode: "NOT_STARTED"}
	}
	p := bts.preparation
	p.mu.Lock()
	defer p.mu.Unlock()
	out := PreparationStatus{Jobs: make([]PreparationJob, 0, len(p.jobs)), QuotaBytes: int64(settings.CurrentFlow().PreparationQuotaMB) << 20, ErrorCode: p.failure}
	out.ReservedBytes, _ = p.reserved()
	for _, j := range p.records() {
		out.Jobs = append(out.Jobs, j.PreparationJob)
	}
	return out
}

// Explicit actions return promptly. Downloads run on the engine lifecycle,
// independently of a preload request, player connection or metadata timeout.
func PrepareEpisode(hashText string, index int, action string) error {
	var hash Hash
	if err := hash.UnmarshalText([]byte(hashText)); err != nil || hash.IsZero() || index < 1 {
		return errors.New("invalid episode")
	}
	if settings.ReadOnly {
		return errors.New("preparation requires writable state")
	}
	if bts == nil || bts.preparation == nil {
		return errors.New("engine not started")
	}
	p := bts.preparation
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.failure != "" {
		return errors.New("preparation state requires repair")
	}
	id := preparationID(hash, index)
	j := p.jobs[id]
	if action == "start" && j == nil {
		if len(p.jobs) >= maxPreparationJobs {
			return errors.New("preparation job limit reached")
		}
		t := GetTorrentInfo(hashText)
		if t == nil {
			return errors.New("torrent not found")
		}
		t.mu.Lock()
		spec := *t.TorrentSpec
		spec.InfoBytes = append([]byte(nil), spec.InfoBytes...)
		t.mu.Unlock()
		metadata, err := lt.ParseTorrentBytes(spec.InfoBytes)
		if err != nil || !metadata.HasMetadata || len(metadata.PieceHashes) != metadata.NumPieces*40 {
			return errors.New("import original v1 torrent or wait for metadata")
		}
		handle := t.LTHandle()
		if handle == nil {
			fresh, e := NewTorrent(&spec, p.bt)
			if e != nil {
				return e
			}
			handle = fresh.LTHandle()
		}
		files, err := handle.Files()
		if err != nil {
			return errors.New("file metadata unavailable")
		}
		var selected *lt.File
		for i := range files {
			if files[i].Index+1 == index {
				selected = &files[i]
				break
			}
		}
		if selected == nil || selected.Size <= 0 {
			return errors.New("file not found")
		}
		j = &preparationRecord{PreparationJob: PreparationJob{ID: id, Hash: hash.HexString(), FileIndex: index, State: "downloading", Length: selected.Size}, Spec: spec, Offset: selected.Offset, PieceLength: metadata.PieceLength, TotalSize: metadata.TotalSize}
		j.First = int(j.Offset / j.PieceLength)
		j.Last = int((j.Offset + j.Length - 1) / j.PieceLength)
		p.jobs[id] = j
		reserved, e := p.reserved()
		if e != nil || reserved > int64(settings.CurrentFlow().PreparationQuotaMB)<<20 {
			delete(p.jobs, id)
			return errors.New("preparation disk quota exceeded")
		}
		if e = p.save(); e != nil {
			delete(p.jobs, id)
			return errors.New("cannot persist preparation job")
		}
		if e = p.configure(hash); e != nil {
			j.State = "error"
			j.ErrorCode = "DISK_WRITE"
			_ = p.save()
			return errors.New("cannot persist cached pieces")
		}
		return nil
	}
	if j == nil {
		return errors.New("preparation job not found")
	}
	previous := j.State
	switch action {
	case "start", "resume":
		reserved, _ := p.reserved()
		if reserved > int64(settings.CurrentFlow().PreparationQuotaMB)<<20 {
			return errors.New("preparation disk quota exceeded")
		}
		j.State = "downloading"
		j.ErrorCode = ""
	case "pause":
		j.State = "paused"
	case "cancel":
		j.State = "cancelled" // retains verified progress until explicit cleanup
	case "remove":
		j.State = "cleaning"
	default:
		return errors.New("unknown preparation action")
	}
	if err := p.save(); err != nil {
		j.State = previous
		return errors.New("cannot persist preparation action")
	}
	if err := p.configure(hash); err != nil {
		return errors.New("preparation storage unavailable")
	}
	return nil
}

func (p *preparationManager) run(stop <-chan struct{}, done chan<- struct{}) {
	defer close(done)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			p.tick()
		}
	}
}

func (p *preparationManager) tick() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.failure != "" {
		return
	}
	changed := false
	cleaning := make(map[Hash]bool)
	diskFailure := make(map[Hash]bool)
	for _, j := range p.jobs {
		if c := torrstor.Global().CacheByHash(j.Spec.InfoHash); c != nil && c.PreparationError() {
			diskFailure[j.Spec.InfoHash] = true
		}
		if j.State == "cleaning" {
			cleaning[j.Spec.InfoHash] = true
		}
	}
	for _, j := range p.records() {
		if j.State == "cleaning" {
			if p.cleanup(j) {
				delete(p.jobs, j.ID)
				_ = p.configure(j.Spec.InfoHash)
				changed = true
			}
			continue
		}
		if cleaning[j.Spec.InfoHash] {
			continue
		}
		t := p.bt.GetTorrent(j.Spec.InfoHash)
		var err error
		if t == nil {
			t, err = NewTorrent(&j.Spec, p.bt)
		}
		if err != nil {
			j.ErrorCode = "ENGINE_WAIT"
			continue
		}
		t.preparationHold.Store(true)
		c := torrstor.Global().CacheByHash(j.Spec.InfoHash)
		if c == nil {
			continue
		}
		if diskFailure[j.Spec.InfoHash] {
			j.State = "error"
			j.ErrorCode = "DISK_WRITE"
			changed = true
			_ = p.configure(j.Spec.InfoHash)
		}
		verified, prefix := c.PreparationProgress(j.Offset, j.Length)
		j.VerifiedBytes, j.ContiguousBytes = verified, prefix
		j.PlaybackReady = verified == j.Length && j.State != "error" // full file, container and arbitrary seeks
		if j.State == "downloading" && verified == j.Length {
			if err := c.FlushPreparation(j.First, j.Last); err != nil {
				j.State = "error"
				j.ErrorCode = "DISK_SYNC"
				j.PlaybackReady = false
			} else {
				j.State = "ready"
				j.ErrorCode = ""
			}
			changed = true
			_ = p.configure(j.Spec.InfoHash)
		}
		c.TickPreparation(t.LTHandle())
	}
	if changed {
		if err := p.save(); err != nil {
			p.failure = "STATE_WRITE"
		}
	}
}

// Cleanup drops the native torrent only when no player owns its cache. The
// native Remove fence joins disk work before files can be removed. Other jobs
// for the same torrent keep their pieces; they are restored on the next tick.
func (p *preparationManager) cleanup(j *preparationRecord) bool {
	c := torrstor.Global().CacheByHash(j.Spec.InfoHash)
	if c != nil && !c.BeginPreparationCleanup() {
		j.ErrorCode = "PLAYBACK_ACTIVE"
		return false
	}
	if t := p.bt.GetTorrent(j.Spec.InfoHash); t != nil {
		t.preparationHold.Store(false)
		p.bt.RemoveTorrent(j.Spec.InfoHash)
		return false
	}
	if c != nil {
		return false
	} // wait for native disk callback Close
	for i := j.First; i <= j.Last; i++ {
		shared := false
		for _, other := range p.jobs {
			if other.ID != j.ID && other.Spec.InfoHash == j.Spec.InfoHash && i >= other.First && i <= other.Last {
				shared = true
				break
			}
		}
		if shared {
			continue
		}
		if err := os.Remove(filepath.Join(p.root, j.Hash, strconv.Itoa(i))); err != nil && !os.IsNotExist(err) {
			j.ErrorCode = "DISK_REMOVE"
			return false
		}
	}
	return true
}
