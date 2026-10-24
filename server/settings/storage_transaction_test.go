package settings

import (
	"encoding/json"
	"errors"
	"os"
	"testing"
)

func TestMigrationRejectsSameLengthDifferentJSON(t *testing.T) {
	equal, err := isByteArraysEqualJson([]byte(`{"x":1}`), []byte(`{"x":2}`))
	if err != nil || equal {
		t.Fatal("same-length corruption accepted")
	}
}
func storageTestDB(t *testing.T) (*TDB, *JsonDB) {
	b := backupTestDB(t)
	oldB, oldJ := globalBboltDB, globalJsonDB
	j := &JsonDB{Path: Path, filenameDelimiter: ".", filenameExtension: ".json", fileMode: 0600, xPathDelimeter: "/"}
	globalBboltDB, globalJsonDB = b, j
	t.Cleanup(func() { globalBboltDB, globalJsonDB = oldB, oldJ })
	setupDatabaseRouting(b, j, false, false)
	if err := SetBTSetsChecked(CloneSettings(BTsets())); err != nil {
		t.Fatal(err)
	}
	SetViewed(&Viewed{Hash: "fixture", FileIndex: 1, TimeCode: 30})
	return b, j
}
func TestStorageMigrationPreservesWAFAndSubsequentWrites(t *testing.T) {
	b, j := storageTestDB(t)
	if err := j.PutChecked("Settings", "waf", []byte(`{"version":1}`)); err != nil {
		t.Fatal(err)
	}
	if err := SetStoragePreferences(map[string]interface{}{"settings": "json", "viewed": "json"}); err != nil {
		t.Fatal(err)
	}
	if !BTsets().StoreSettingsInJson || !BTsets().StoreViewedInJson || b.Get("Settings", "BitTorr") != nil || len(b.List("Viewed")) != 0 {
		t.Fatal("storage route not committed")
	}
	if len(j.Get("Settings", "waf")) == 0 || len(ListViewed("fixture")) != 1 {
		t.Fatal("unrelated config or viewed data lost")
	}
	MarkViewed("fixture", 2)
	if len(ListViewed("fixture")) != 2 {
		t.Fatal("new route not used")
	}
	if err := SetStoragePreferences(map[string]interface{}{"settings": "bbolt", "viewed": "bbolt"}); err != nil {
		t.Fatal(err)
	}
	if len(ListViewed("fixture")) != 2 || len(j.Get("Settings", "waf")) == 0 {
		t.Fatal("reverse migration lost data")
	}
}
func TestStorageJournalRecoversInterruptedMigration(t *testing.T) {
	b, j := storageTestDB(t)
	beforeB, _ := b.StorageSnapshot()
	beforeJ, _ := j.StorageSnapshot()
	raw, _ := json.Marshal(storageJournal{1, beforeB, beforeJ})
	if err := os.WriteFile(storageJournalPath(), raw, 0600); err != nil {
		t.Fatal(err)
	}
	_ = b.PutChecked("Viewed", "fixture", []byte(`{"9":50}`))
	if err := recoverStorageTransaction(b, j); err != nil {
		t.Fatal(err)
	}
	after, _ := b.StorageSnapshot()
	expected, _ := json.Marshal(beforeB)
	actual, _ := json.Marshal(after)
	if equal, _ := isByteArraysEqualJson(expected, actual); !equal {
		t.Fatal("original not restored")
	}
	if _, err := os.Stat(storageJournalPath()); !os.IsNotExist(err) {
		t.Fatal("journal not retired")
	}
}

type failedStorage struct {
	TorrServerDB
	storageBackend
	fail bool
}

func (f *failedStorage) RestoreStorage(data storageData) error {
	err := f.storageBackend.RestoreStorage(data)
	if f.fail {
		f.fail = false
		return errors.New("injected post-write failure")
	}
	return err
}
func TestStorageMigrationRollsBackBothBackendsAndPreferences(t *testing.T) {
	b, j := storageTestDB(t)
	original := SettingsRevision(BTsets())
	beforeB, _ := b.StorageSnapshot()
	beforeJ, _ := j.StorageSnapshot()
	globalJsonDB = &failedStorage{TorrServerDB: j, storageBackend: j, fail: true}
	err := SetStoragePreferences(map[string]interface{}{"settings": "json", "viewed": "json"})
	if err == nil {
		t.Fatal("injected failure ignored")
	}
	afterB, _ := b.StorageSnapshot()
	afterJ, _ := j.StorageSnapshot()
	for _, pair := range [][2]storageData{{beforeB, afterB}, {beforeJ, afterJ}} {
		a, _ := json.Marshal(pair[0])
		b, _ := json.Marshal(pair[1])
		if equal, _ := isByteArraysEqualJson(a, b); !equal {
			t.Fatal("rollback left changed data")
		}
	}
	if original != SettingsRevision(BTsets()) || StorageRecoveryPending() {
		t.Fatal("preferences or recovery state not rolled back")
	}
}
