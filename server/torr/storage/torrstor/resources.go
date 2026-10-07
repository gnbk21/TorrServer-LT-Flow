package torrstor

import (
	"server/diagnostics"
	"server/settings"
	"sort"
	"sync"
	"time"
)

type ResourceStatus struct {
	BudgetBytes         int64     `json:"budget_bytes"`
	WarmBudgetBytes     int64     `json:"warm_budget_bytes"`
	MemoryPressure      bool      `json:"memory_pressure"`
	ProtectedOvercommit bool      `json:"protected_overcommit"`
	BackgroundLimited   bool      `json:"background_limited"`
	RAMResidentBytes    int64     `json:"ram_resident_bytes"`
	DiskCacheBytes      int64     `json:"disk_cache_bytes"`
	SampledAt           time.Time `json:"sampled_at"`
}

var resourceState struct {
	sync.Mutex
	status ResourceStatus
}

func Resources() ResourceStatus {
	resourceState.Lock()
	defer resourceState.Unlock()
	return resourceState.status
}

// ReconcileResources changes budgets, not ownership. Existing immediate reader,
// preload and container pins always win; overcommit is visible, never hidden.
func (s *Storage) ReconcileResources() {
	resourceState.Lock()
	defer resourceState.Unlock()
	if time.Since(resourceState.status.SampledAt) < time.Second {
		return
	}
	f := settings.CurrentFlow()
	mem := diagnostics.Memory()
	budget := int64(f.GlobalCacheBudgetMB) << 20
	if budget == 0 {
		budget = 4 << 30
		if mem.SystemTotalBytes > 0 {
			budget = min(budget, int64(mem.SystemTotalBytes/4))
		}
	}
	budget = max(64<<20, budget)
	if mem.Pressure {
		budget = max(64<<20, budget/2)
	}
	out := ResourceStatus{BudgetBytes: budget, WarmBudgetBytes: min(budget/4, int64(f.WarmCacheBudgetMB)<<20), MemoryPressure: mem.Pressure, SampledAt: time.Now()}
	s.mu.RLock()
	caches := make([]*Cache, 0, len(s.caches))
	for _, c := range s.caches {
		caches = append(caches, c)
	}
	s.mu.RUnlock()
	sort.Slice(caches, func(i, j int) bool { return caches[i].StorageID < caches[j].StorageID })
	active := 0
	for _, c := range caches {
		if c.StreamingReaders() > 0 && !c.diskBacked() {
			active++
		}
	}
	warmCount := 0
	for _, c := range caches {
		if c.StreamingReaders() == 0 && !c.diskBacked() {
			warmCount++
		}
	}
	if warmCount == 0 {
		out.WarmBudgetBytes = 0
	}
	activeBudget := budget - out.WarmBudgetBytes
	var protected int64
	for _, c := range caches {
		disk := c.diskBacked()
		readers := c.StreamingReaders()
		if disk {
			out.DiskCacheBytes += c.Filled()
			c.resourceBudget.Store(0)
			c.backgroundLimited.Store(mem.Pressure)
			continue
		}
		out.RAMResidentBytes += c.Filled()
		limit := max(c.PieceLength, activeBudget/max(1, int64(active)))
		if readers == 0 {
			limit = max(c.PieceLength, out.WarmBudgetBytes/max(1, int64(warmCount)))
			if mem.Pressure || f.WarmCacheBudgetMB == 0 {
				c.ClearWarmReserve()
				limit = max(1, c.PieceLength)
			}
		}
		c.resourceBudget.Store(limit)
		protected += c.streamingReserve()
		c.backgroundLimited.Store(mem.Pressure)
		go c.evictIfOverCapacity()
	}
	out.ProtectedOvercommit = protected > budget
	out.BackgroundLimited = mem.Pressure || out.ProtectedOvercommit
	if out.BackgroundLimited {
		for _, c := range caches {
			c.backgroundLimited.Store(true)
		}
	}
	resourceState.status = out
}

func (c *Cache) diskBacked() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.diskRoot != "" || useDisk()
}
