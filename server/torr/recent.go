package torr

import (
	"crypto/sha256"
	"fmt"
	"server/settings"
	"server/torr/storage/torrstor"
	"server/utils"
	"sync/atomic"
	"time"
)

type recentOpenEvent struct {
	hash    string
	index   int
	session string
	at      time.Time
}
type recentOpenQueue struct{ events []recentOpenEvent }
type recentDelivery struct {
	id     string
	active int
	seen   time.Time
}

var recentSequence atomic.Uint64

// Minimal session grouping when ordinary Flow measurements are disabled.
// This uses the existing reader group and warm timeout; it collects no rates,
// probes or decoded positions and retains only hashed group keys in memory.
func (t *Torrent) beginRecentDelivery(index int, group string) (string, func()) {
	key := sha256.Sum256([]byte(fmt.Sprint(index, ":", group)))
	t.recentDeliveryMu.Lock()
	if t.recentDeliveries == nil {
		t.recentDeliveries = map[[32]byte]*recentDelivery{}
	}
	now := time.Now()
	row := t.recentDeliveries[key]
	if row == nil {
		if len(t.recentDeliveries) >= 128 {
			for k, v := range t.recentDeliveries {
				if v.active == 0 {
					delete(t.recentDeliveries, k)
					break
				}
			}
		}
		if len(t.recentDeliveries) >= 128 {
			t.recentDeliveryMu.Unlock()
			return "", func() {}
		}
		row = &recentDelivery{}
		t.recentDeliveries[key] = row
	}
	if row.active == 0 && (row.id == "" || now.Sub(row.seen) > time.Duration(settings.CurrentFlow().WarmSessionTimeoutSec)*time.Second) {
		row.id = fmt.Sprint("recent-", recentSequence.Add(1))
	}
	row.active++
	row.seen = now
	id := row.id
	t.recentDeliveryMu.Unlock()
	return id, func() { t.recentDeliveryMu.Lock(); row.active--; row.seen = time.Now(); t.recentDeliveryMu.Unlock() }
}

// First-byte observation queues a tiny immutable event. Database I/O belongs
// to the existing watcher, never the HTTP media delivery callback.
func (t *Torrent) queueRecentOpen(index int, group, session string) {
	if !settings.CurrentFlow().RecentPlaybackHistory || settings.ReadOnly || group == torrstor.ProbeReaderGroup {
		return
	}
	file := t.fileByID(index)
	if file == nil || utils.GetMimeType(file.Path) != "video/*" {
		return
	}
	if session == "" {
		session = t.FlowSessionID(index, group)
	}
	event := recentOpenEvent{hash: t.Hash().HexString(), index: index, session: session, at: time.Now().UTC()}
	for {
		old := t.recentOpen.Load()
		next := &recentOpenQueue{}
		if old != nil {
			for _, e := range old.events {
				if e.index == event.index && e.session == event.session {
					return
				}
			}
			if len(old.events) >= 64 {
				return
			}
			next.events = append(next.events, old.events...)
		}
		next.events = append(next.events, event)
		if t.recentOpen.CompareAndSwap(old, next) {
			return
		}
	}
}
func (t *Torrent) flushRecentOpen() {
	if queue := t.recentOpen.Swap(nil); queue != nil {
		for _, event := range queue.events {
			settings.RecordRecentOpen(event.hash, event.index, event.session, event.at)
		}
	}
}
