package settings

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

const recentLimit = 200

type RecentItem struct {
	Hash         string    `json:"hash"`
	FileIndex    int       `json:"file_index"`
	LastOpenedAt time.Time `json:"last_opened_at"`
}
type ViewedMetadata struct {
	Hash              string     `json:"hash"`
	FileIndex         int        `json:"file_index"`
	PositionSource    string     `json:"position_source,omitempty"`
	PositionUpdatedAt *time.Time `json:"position_updated_at,omitempty"`
	Completed         *bool      `json:"completed,omitempty"`
}
type recentDocument struct {
	SchemaVersion int          `json:"schema_version"`
	Items         []RecentItem `json:"items"`
}
type viewedMetadataDocument struct {
	SchemaVersion int              `json:"schema_version"`
	Items         []ViewedMetadata `json:"items"`
}

var recentMu sync.Mutex
var recentSessions = map[[32]byte]time.Time{}
var recentPrunedAt atomic.Int64

// The existing configuration worker owns expiry maintenance. Reads remain
// observational, and disabled collection does not retain expired opens forever.
func PruneRecent(at time.Time) {
	if ReadOnly {
		return
	}
	previous := recentPrunedAt.Load()
	if previous > 0 && at.Unix()-previous < 3600 {
		return
	}
	if !recentPrunedAt.CompareAndSwap(previous, at.Unix()) {
		return
	}
	dbMigrationLock.RLock()
	defer dbMigrationLock.RUnlock()
	recentMu.Lock()
	defer recentMu.Unlock()
	doc, err := readRecentDocumentLocked()
	if err != nil || tdb == nil {
		return
	}
	items := []RecentItem{}
	for _, row := range doc.Items {
		if recentIdentity(row.Hash, row.FileIndex) && row.LastOpenedAt.After(at.Add(-30*24*time.Hour)) && !row.LastOpenedAt.After(at.Add(time.Minute)) {
			items = append(items, row)
		}
	}
	if len(items) != len(doc.Items) {
		doc.Items = items
		_ = putRecentDocumentLocked(doc, "FlowRecent")
	}
	for key, seen := range recentSessions {
		if seen.Before(at.Add(-30 * 24 * time.Hour)) {
			delete(recentSessions, key)
		}
	}
}

func recentIdentity(hash string, index int) bool {
	raw, err := hex.DecodeString(hash)
	return err == nil && (len(raw) == 20 || len(raw) == 32) && index > 0
}
func readRecentDocumentLocked() (recentDocument, error) {
	out := recentDocument{SchemaVersion: 1, Items: []RecentItem{}}
	if tdb == nil {
		return out, nil
	}
	raw := tdb.Get("FlowRecent", "v1")
	if len(raw) == 0 {
		return out, nil
	}
	if len(raw) > 128<<10 || json.Unmarshal(raw, &out) != nil || out.SchemaVersion != 1 || len(out.Items) > recentLimit {
		return recentDocument{}, errors.New("recent history is invalid")
	}
	return out, nil
}
func putRecentDocumentLocked(value any, bucket string) error {
	if ReadOnly {
		return errors.New("database is read-only")
	}
	if tdb == nil {
		return errors.New("database is unavailable")
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if checked, ok := tdb.(interface {
		PutChecked(string, string, []byte) error
	}); ok {
		return checked.PutChecked(bucket, "v1", raw)
	}
	tdb.Set(bucket, "v1", raw)
	return nil
}
func RecordRecentOpen(hash string, index int, session string, at time.Time) {
	if !CurrentFlow().RecentPlaybackHistory || ReadOnly || !recentIdentity(hash, index) {
		return
	}
	dbMigrationLock.RLock()
	defer dbMigrationLock.RUnlock()
	recentMu.Lock()
	defer recentMu.Unlock()
	if tdb == nil {
		return
	}
	key := sha256.Sum256([]byte(fmt.Sprint(hash, ":", index, ":", session)))
	if previous, ok := recentSessions[key]; ok && (session != "" || at.Sub(previous) < time.Minute) {
		return
	}
	document, err := readRecentDocumentLocked()
	if err != nil {
		return
	}
	items := []RecentItem{{Hash: hash, FileIndex: index, LastOpenedAt: at.UTC()}}
	for _, row := range document.Items {
		if len(items) == recentLimit {
			break
		}
		if row.Hash == hash && row.FileIndex == index {
			continue
		}
		if row.LastOpenedAt.After(at.Add(-30*24*time.Hour)) && recentIdentity(row.Hash, row.FileIndex) {
			items = append(items, row)
		}
	}
	document.Items = items
	if putRecentDocumentLocked(document, "FlowRecent") == nil {
		if len(recentSessions) >= 1000 {
			for k := range recentSessions {
				delete(recentSessions, k)
				break
			}
		}
		recentSessions[key] = at
	}
}
func ListRecent() (items []RecentItem, err error) {
	items = []RecentItem{}
	if !CurrentFlow().RecentPlaybackHistory {
		return items, nil
	}
	dbMigrationLock.RLock()
	defer dbMigrationLock.RUnlock()
	recentMu.Lock()
	defer recentMu.Unlock()
	doc, err := readRecentDocumentLocked()
	if err != nil {
		return items, err
	}
	now := time.Now()
	for _, row := range doc.Items {
		if recentIdentity(row.Hash, row.FileIndex) && !row.LastOpenedAt.After(now.Add(time.Minute)) && row.LastOpenedAt.After(now.Add(-30*24*time.Hour)) {
			items = append(items, row)
		}
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].LastOpenedAt.After(items[j].LastOpenedAt) })
	return items, nil
}
func RemoveRecent(hash string, index int, clear bool) error {
	dbMigrationLock.RLock()
	defer dbMigrationLock.RUnlock()
	recentMu.Lock()
	defer recentMu.Unlock()
	doc, err := readRecentDocumentLocked()
	if err != nil && !clear {
		return err
	}
	if clear {
		doc = recentDocument{SchemaVersion: 1, Items: []RecentItem{}}
	} else {
		out := []RecentItem{}
		for _, row := range doc.Items {
			if row.Hash != hash || (index > 0 && row.FileIndex != index) {
				out = append(out, row)
			}
		}
		doc.Items = out
	}
	return putRecentDocumentLocked(doc, "FlowRecent")
}

// Caller already holds dbMigrationLock; legacy Viewed encodings stay untouched.
func viewedMetadataLocked() viewedMetadataDocument {
	out := viewedMetadataDocument{SchemaVersion: 1, Items: []ViewedMetadata{}}
	if tdb == nil {
		return out
	}
	raw := tdb.Get("FlowViewedMetadata", "v1")
	if len(raw) > 128<<10 || len(raw) > 0 && (json.Unmarshal(raw, &out) != nil || out.SchemaVersion != 1 || len(out.Items) > recentLimit) {
		return viewedMetadataDocument{SchemaVersion: 1, Items: []ViewedMetadata{}}
	}
	return out
}
func updateViewedMetadataLocked(v *Viewed, position bool) {
	if ReadOnly || tdb == nil {
		return
	}
	recentMu.Lock()
	defer recentMu.Unlock()
	doc := viewedMetadataLocked()
	row := ViewedMetadata{Hash: v.Hash, FileIndex: v.FileIndex}
	items := []ViewedMetadata{}
	for _, item := range doc.Items {
		if item.Hash == v.Hash && item.FileIndex == v.FileIndex {
			row = item
		} else {
			items = append(items, item)
		}
	}
	if position {
		now := time.Now().UTC()
		row.PositionSource = "client_reported"
		if v.PositionSource == "internal_player" {
			row.PositionSource = "internal_player"
		}
		row.PositionUpdatedAt = &now
	}
	if v.Completed != nil {
		completed := *v.Completed
		row.Completed = &completed
	}
	doc.Items = append([]ViewedMetadata{row}, items...)
	doc.Items = doc.Items[:min(recentLimit, len(doc.Items))]
	_ = putRecentDocumentLocked(doc, "FlowViewedMetadata")
}
func removeViewedMetadataLocked(hash string, index int) {
	if ReadOnly || tdb == nil {
		return
	}
	recentMu.Lock()
	defer recentMu.Unlock()
	doc := viewedMetadataLocked()
	out := []ViewedMetadata{}
	for _, row := range doc.Items {
		if row.Hash != hash || index >= 0 && row.FileIndex != index {
			out = append(out, row)
		}
	}
	doc.Items = out
	_ = putRecentDocumentLocked(doc, "FlowViewedMetadata")
}
