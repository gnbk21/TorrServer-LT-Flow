package torrstor

import (
	"server/flow"
	"server/settings"
	"time"
)

func storageOperationStart() time.Time {
	f := settings.CurrentFlow()
	if !f.Enabled || !f.MetricsEnabled {
		return time.Time{}
	}
	return time.Now()
}

func (c *Cache) DrainStorageEvidence() ([]flow.HistoryEvent, uint64) {
	return c.storageEvidence.Drain()
}
