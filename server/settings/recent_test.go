package settings

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

func recentTestDB(t *testing.T) *memDB {
	db := withViewedDB(t, true)
	sets := NewDefaultConfig()
	sets.TrackTimecode = true
	StoreBTsets(sets)
	recentMu.Lock()
	old := recentSessions
	recentSessions = map[[32]byte]time.Time{}
	recentMu.Unlock()
	t.Cleanup(func() { recentMu.Lock(); recentSessions = old; recentMu.Unlock() })
	return db
}
func TestRecentRetentionDebounceDisableAndRemoval(t *testing.T) {
	recentTestDB(t)
	now := time.Now()
	hash := fmt.Sprintf("%040x", 1)
	RecordRecentOpen(hash, 1, "session", now)
	RecordRecentOpen(hash, 1, "session", now.Add(time.Second))
	rows, err := ListRecent()
	if err != nil || len(rows) != 1 || !rows[0].LastOpenedAt.Equal(now) {
		t.Fatal(rows, err)
	}
	for i := 2; i <= 210; i++ {
		RecordRecentOpen(fmt.Sprintf("%040x", i), 1, "session", now)
	}
	rows, _ = ListRecent()
	if len(rows) != 200 {
		t.Fatal("retention", len(rows))
	}
	current := CloneSettings(BTsets())
	current.Flow.RecentPlaybackHistory = false
	StoreBTsets(current)
	RecordRecentOpen(hash, 2, "new", now)
	rows, _ = ListRecent()
	if len(rows) != 0 {
		t.Fatal("disabled history visible")
	}
	current.Flow.RecentPlaybackHistory = true
	StoreBTsets(current)
	if err := RemoveRecent(fmt.Sprintf("%040x", 210), -1, false); err != nil {
		t.Fatal(err)
	}
	rows, _ = ListRecent()
	if len(rows) != 199 {
		t.Fatal("remove", len(rows))
	}
	if err := RemoveRecent("", 0, true); err != nil {
		t.Fatal(err)
	}
	rows, _ = ListRecent()
	if len(rows) != 0 {
		t.Fatal("clear", rows)
	}
}
func TestRecentExpiresAndDoesNotInferCompletion(t *testing.T) {
	db := recentTestDB(t)
	hash := fmt.Sprintf("%064x", 1)
	RecordRecentOpen(hash, 1, "old", time.Now().Add(-31*24*time.Hour))
	rows, _ := ListRecent()
	if len(rows) != 0 {
		t.Fatal("old history remains")
	}
	db.Set("Viewed", hash, []byte(`{"1":42,"2":0}`))
	viewed := ListViewed(hash)
	for _, row := range viewed {
		if row.Completed != nil || row.PositionUpdatedAt != nil {
			t.Fatal("fabricated provenance", row)
		}
		if row.FileIndex == 1 && row.PositionSource != "legacy_saved" {
			t.Fatal("missing legacy label")
		}
	}
	SetViewed(&Viewed{Hash: hash, FileIndex: 1, TimeCode: 50, PositionSource: "internal_player"})
	complete := true
	SetViewed(&Viewed{Hash: hash, FileIndex: 1, Completed: &complete})
	for _, row := range ListViewed(hash) {
		if row.FileIndex == 1 && (row.TimeCode != 50 || row.PositionSource != "internal_player" || row.PositionUpdatedAt == nil || row.Completed == nil || !*row.Completed) {
			t.Fatal("completion erased position", row)
		}
	}
	var raw map[int]float64
	if json.Unmarshal(db.Get("Viewed", hash), &raw) != nil || raw[1] != 50 {
		t.Fatal("legacy encoding changed")
	}
	sets := CloneSettings(BTsets())
	sets.TrackTimecode = false
	StoreBTsets(sets)
	SetViewed(&Viewed{Hash: hash, FileIndex: 2, TimeCode: 10, PositionSource: "internal_player"})
	for _, row := range ListViewed(hash) {
		if row.FileIndex == 2 && (row.TimeCode != 0 || row.PositionUpdatedAt != nil) {
			t.Fatal("silently enabled tracking")
		}
	}
}

func TestExpiredRecentIsPhysicallyPrunedWithoutPlayback(t *testing.T) {
	db := recentTestDB(t)
	old := recentPrunedAt.Swap(0)
	t.Cleanup(func() { recentPrunedAt.Store(old) })
	hash := fmt.Sprintf("%040x", 1)
	RecordRecentOpen(hash, 1, "old", time.Now().Add(-31*24*time.Hour))
	PruneRecent(time.Now())
	var document recentDocument
	if json.Unmarshal(db.Get("FlowRecent", "v1"), &document) != nil || len(document.Items) != 0 {
		t.Fatal("expired identity persisted", document)
	}
}
func TestNewPresentationSettingsDefaultsAndHotApplication(t *testing.T) {
	var old FlowSettings
	if err := json.Unmarshal([]byte(`{"Enabled":true}`), &old); err != nil {
		t.Fatal(err)
	}
	if !old.RecentPlaybackHistory || old.PrometheusEnabled {
		t.Fatal("migration defaults")
	}
	if err := json.Unmarshal([]byte(`{"RecentPlaybackHistory":false,"PrometheusEnabled":true}`), &old); err != nil || old.RecentPlaybackHistory || !old.PrometheusEnabled {
		t.Fatal("explicit settings lost", err)
	}
	a := NewDefaultConfig()
	b := CloneSettings(a)
	b.Flow.RecentPlaybackHistory = false
	b.Flow.PrometheusEnabled = true
	if NeedsEngineRestart(a, b) {
		t.Fatal("read-only presentation controls require restart")
	}
	metadata := SettingsMetadata()
	for _, key := range []string{"Flow.RecentPlaybackHistory", "Flow.PrometheusEnabled"} {
		if metadata.Fields[key].RestartRequired || metadata.Fields[key].Type != "boolean" {
			t.Fatal("metadata drift", key)
		}
	}
}
