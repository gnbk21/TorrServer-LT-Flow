package torrstor

import (
	"time"

	"server/flow"
	"server/settings"
)

type NextEpisodeStatus struct {
	FileIndex     int    `json:"file_index"`
	Automatic     bool   `json:"automatic"`
	State         string `json:"state"`
	Reason        string `json:"reason"`
	BudgetBytes   int64  `json:"budget_bytes"`
	VerifiedBytes int64  `json:"verified_bytes"`
}

type nextEpisodeWarmup struct {
	NextEpisodeStatus
	pieces   []int
	until    time.Time
	eligible bool
	done     bool
}

func (c *Cache) ClearNextEpisodeWarmup() {
	if c == nil {
		return
	}
	c.nextMu.Lock()
	c.nextWarmup = nextEpisodeWarmup{}
	if c.storage != nil {
		c.storage.nextMu.Lock()
		if c.storage.nextOwner == c {
			c.storage.nextOwner = nil
			c.storage.nextUntil = time.Time{}
		}
		c.storage.nextMu.Unlock()
	}
	c.nextMu.Unlock()
}

// The watcher renews an eligible plan; priority ticks cannot keep abandoned
// work alive. Reusing a target preserves completion instead of fetching again
// whenever an unprotected warm piece is naturally evicted.
func (c *Cache) SetNextEpisodeWarmup(fileID int, offset, length int64, automatic, eligible bool, reason string) {
	pieces, budget := flow.WarmupPieces(offset, length, c.PieceLength, c.NumPieces)
	c.nextMu.Lock()
	defer c.nextMu.Unlock()
	if c.nextWarmup.FileIndex != fileID {
		c.nextWarmup = nextEpisodeWarmup{NextEpisodeStatus: NextEpisodeStatus{FileIndex: fileID, Automatic: automatic}, pieces: pieces}
	}
	w := &c.nextWarmup
	w.BudgetBytes, w.eligible, w.Reason = budget, eligible, reason
	w.until = time.Now().Add(3 * time.Second)
	if c.storage != nil {
		c.storage.nextMu.Lock()
		if owner := c.storage.nextOwner; owner != nil && owner != c && time.Now().Before(c.storage.nextUntil) {
			w.eligible = false
			w.State = "waiting"
			w.Reason = "OTHER_WARMUP"
		} else {
			c.storage.nextOwner = c
			c.storage.nextUntil = w.until
		}
		c.storage.nextMu.Unlock()
	}
	if len(w.pieces) == 0 {
		w.eligible = false
		w.State = "unavailable"
		w.Reason = "PIECE_BUDGET"
	}
}

func (c *Cache) nextEpisodeDemand(blocked bool) []int {
	c.nextMu.Lock()
	defer c.nextMu.Unlock()
	w := &c.nextWarmup
	if w.FileIndex == 0 {
		return nil
	}
	verified := int64(0)
	missing := make([]int, 0, len(w.pieces))
	for _, p := range w.pieces {
		if c.Have(p) {
			verified += c.PieceLength
		} else {
			missing = append(missing, p)
		}
	}
	w.VerifiedBytes = verified
	if len(w.pieces) == 0 {
		return nil
	}
	if len(missing) == 0 {
		w.done = true
		w.State = "ready"
		w.Reason = "VERIFIED"
		return nil
	}
	if w.done {
		w.State = "evicted"
		w.Reason = "CACHE_REUSED"
		return nil
	}
	w.State = "waiting"
	if !settings.CurrentFlow().Enabled || !settings.CurrentFlow().NextEpisodeWarmup || time.Now().After(w.until) {
		w.Reason = "DISABLED_OR_EXPIRED"
		return nil
	}
	if !w.eligible {
		return nil
	}
	if blocked || c.flowWaiting.Load() > 0 || c.flowReconnect.Load() {
		w.Reason = "PLAYBACK_WAIT"
		return nil
	}
	if c.storage != nil {
		c.storage.mu.RLock()
		waiting := false
		for _, other := range c.storage.caches {
			waiting = waiting || other.flowWaiting.Load() > 0 || other.flowReconnect.Load()
		}
		c.storage.mu.RUnlock()
		if waiting {
			w.Reason = "PLAYBACK_WAIT"
			return nil
		}
	}
	if c.backgroundLimited.Load() {
		w.Reason = "RESOURCE_PRESSURE"
		return nil
	}
	// Use the configured/coordinated base budget, never capacity()'s growing
	// protected-reader reserve. Warmup must not enlarge either budget.
	budget := globalCacheSize()
	if limit := c.resourceBudget.Load(); limit > 0 {
		budget = min(budget, limit)
	}
	if budget <= 0 || c.Filled()+int64(len(missing))*c.PieceLength > budget {
		w.Reason = "NO_SPARE_CACHE"
		return nil
	}
	w.State, w.Reason = "warming", "SPARE_CAPACITY"
	return missing
}

func (c *Cache) NextEpisodeWarmupStatus() NextEpisodeStatus {
	if c == nil {
		return NextEpisodeStatus{State: "idle"}
	}
	c.nextMu.Lock()
	defer c.nextMu.Unlock()
	s := c.nextWarmup.NextEpisodeStatus
	if s.State == "" {
		s.State = "idle"
	}
	return s
}
