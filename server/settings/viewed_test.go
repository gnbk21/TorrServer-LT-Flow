package settings

import (
	"path/filepath"
	"sort"
	"sync"
	"testing"

	bolt "go.etcd.io/bbolt"
)

// memDB is an in-memory TorrServerDB for tests.
type memDB struct{ data map[string]map[string][]byte }

func newMemDB() *memDB { return &memDB{data: map[string]map[string][]byte{}} }

func (m *memDB) CloseDB() {}
func (m *memDB) Get(xPath, name string) []byte {
	return m.data[xPath][name]
}
func (m *memDB) Set(xPath, name string, value []byte) {
	if m.data[xPath] == nil {
		m.data[xPath] = map[string][]byte{}
	}
	m.data[xPath][name] = value
}
func (m *memDB) List(xPath string) []string {
	var keys []string
	for k := range m.data[xPath] {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
func (m *memDB) Rem(xPath, name string) { delete(m.data[xPath], name) }
func (m *memDB) Clear(xPath string)     { delete(m.data, xPath) }

func withViewedDB(t *testing.T, track bool) *memDB {
	t.Helper()
	db := newMemDB()
	oldDB, oldSets := tdb, BTsets()
	tdb = db
	StoreBTsets(&BTSets{TrackTimecode: track})
	t.Cleanup(func() {
		tdb = oldDB
		StoreBTsets(oldSets)
	})
	return db
}

func timecodes(hash string) map[int]float64 {
	out := map[int]float64{}
	for _, v := range ListViewed(hash) {
		out[v.FileIndex] = v.TimeCode
	}
	return out
}

func TestSetViewedStoresTimecodeOnlyWhenTracking(t *testing.T) {
	withViewedDB(t, true)
	SetViewed(&Viewed{Hash: "h", FileIndex: 1, TimeCode: 42.5})
	if got := timecodes("h")[1]; got != 42.5 {
		t.Fatalf("tracked timecode = %v, want 42.5", got)
	}

	withViewedDB(t, false)
	SetViewed(&Viewed{Hash: "h", FileIndex: 1, TimeCode: 42.5})
	if got := timecodes("h")[1]; got != 0 {
		t.Fatalf("untracked timecode = %v, want 0", got)
	}
}

func TestMarkViewedKeepsSavedTimecode(t *testing.T) {
	withViewedDB(t, true)
	SetViewed(&Viewed{Hash: "h", FileIndex: 2, TimeCode: 600})
	MarkViewed("h", 2)
	MarkViewed("h", 3)
	got := timecodes("h")
	if got[2] != 600 {
		t.Fatalf("MarkViewed reset the saved timecode: %v", got[2])
	}
	if tc, ok := got[3]; !ok || tc != 0 {
		t.Fatalf("MarkViewed did not add file 3: %v", got)
	}
}

func TestConcurrentViewedUpdatesKeepEveryFile(t *testing.T) {
	db, err := bolt.Open(filepath.Join(t.TempDir(), "config.db"), 0o600, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	withViewedDB(t, true)
	tdb = &TDB{db: db}

	const files = 24
	for file := 1; file <= files; file++ {
		MarkViewed("h", files+file)
	}

	var writers sync.WaitGroup
	start := make(chan struct{})
	for file := 1; file <= files; file++ {
		writers.Go(func() {
			<-start
			MarkViewed("h", file)
		})
		writers.Go(func() {
			<-start
			SetViewed(&Viewed{Hash: "h", FileIndex: file, TimeCode: float64(file)})
		})
		writers.Go(func() {
			<-start
			RemViewed(&Viewed{Hash: "h", FileIndex: files + file})
		})
	}
	close(start)
	writers.Wait()

	got := timecodes("h")
	if len(got) != files {
		t.Errorf("viewed records = %d, want %d", len(got), files)
	}
	for file := 1; file <= files; file++ {
		if got[file] != float64(file) {
			t.Errorf("file %d timecode = %v, want %d", file, got[file], file)
		}
	}
}

func TestViewedReadsLegacyRecords(t *testing.T) {
	db := withViewedDB(t, true)
	db.Set("Viewed", "old", []byte(`{"1":{},"4":{}}`))
	got := timecodes("old")
	if len(got) != 2 || got[1] != 0 || got[4] != 0 {
		t.Fatalf("legacy record = %v", got)
	}
	// A new mark rewrites the record in the timecode format.
	SetViewed(&Viewed{Hash: "old", FileIndex: 5, TimeCode: 7})
	if got := timecodes("old"); len(got) != 3 || got[5] != 7 {
		t.Fatalf("after upgrade = %v", got)
	}
}

func TestListViewedAllSkipsEmptyRecords(t *testing.T) {
	db := withViewedDB(t, false)
	db.Set("Viewed", "a", []byte{})
	MarkViewed("b", 1)
	if got := ListViewed(""); len(got) != 1 || got[0].Hash != "b" {
		t.Fatalf("ListViewed(\"\") = %+v, want the one mark of b", got)
	}
}

func TestRemViewed(t *testing.T) {
	withViewedDB(t, false)
	MarkViewed("h", 1)
	MarkViewed("h", 2)
	RemViewed(&Viewed{Hash: "h", FileIndex: 1})
	if got := timecodes("h"); len(got) != 1 || got[2] != 0 {
		t.Fatalf("after rem one = %v", got)
	}
	RemViewed(&Viewed{Hash: "h", FileIndex: -1})
	if got := ListViewed("h"); len(got) != 0 {
		t.Fatalf("after rem all = %v", got)
	}
}
