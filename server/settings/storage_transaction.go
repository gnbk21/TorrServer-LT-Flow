package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"server/diagnostics"

	bolt "go.etcd.io/bbolt"
)

// Only BitTorr and Viewed belong to this operation. In particular, WAF and
// GStreamer keep their existing JSON storage independently of these choices.
type storageData map[string]map[string]json.RawMessage
type storageJournal struct {
	Schema int         `json:"schema"`
	Bolt   storageData `json:"bolt"`
	JSON   storageData `json:"json"`
}
type storageBackend interface {
	StorageSnapshot() (storageData, error)
	RestoreStorage(storageData) error
}

func storageJournalPath() string { return filepath.Join(Path, "flow-storage-transaction.json") }

func (v *TDB) StorageSnapshot() (storageData, error) {
	out := storageData{"Settings": {}, "Viewed": {}}
	total := 0
	err := v.db.View(func(tx *bolt.Tx) error {
		for path, rows := range out {
			b := tx.Bucket([]byte(path))
			if b == nil {
				continue
			}
			if err := b.ForEach(func(k, value []byte) error {
				if path == "Settings" && string(k) != "BitTorr" {
					return nil
				}
				if value == nil || !json.Valid(value) {
					return errors.New("invalid storage record")
				}
				total += len(k) + len(value)
				if total > BackupLimit {
					return errors.New("storage snapshot exceeds recovery limit")
				}
				rows[string(k)] = append(json.RawMessage(nil), value...)
				return nil
			}); err != nil {
				return err
			}
		}
		return nil
	})
	return out, err
}
func (v *TDB) RestoreStorage(data storageData) error {
	return v.db.Update(func(tx *bolt.Tx) error {
		for _, path := range []string{"Settings", "Viewed"} {
			b, err := tx.CreateBucketIfNotExists([]byte(path))
			if err != nil {
				return err
			}
			if path == "Settings" {
				if err = b.Delete([]byte("BitTorr")); err != nil {
					return err
				}
			} else {
				c := b.Cursor()
				for k, _ := c.First(); k != nil; k, _ = c.Next() {
					if err = c.Delete(); err != nil {
						return err
					}
				}
			}
			for k, value := range data[path] {
				if err = b.Put([]byte(k), value); err != nil {
					return err
				}
			}
		}
		return nil
	})
}
func (v *JsonDB) StorageSnapshot() (storageData, error) {
	out := storageData{"Settings": {}, "Viewed": {}}
	for _, path := range []string{"Settings", "Viewed"} {
		filename, _ := v.xPathToFilename(path)
		if info, err := os.Lstat(filepath.Join(v.Path, filename)); err == nil && (!info.Mode().IsRegular() || info.Size() > BackupLimit) {
			return nil, errors.New("storage snapshot is not a bounded regular file")
		}
		v.lock(filename)
		root, err := v.readJsonFileAsMap(filename)
		v.unlock(filename)
		if err != nil {
			return nil, err
		}
		for k, value := range root {
			if path == "Settings" && k != "BitTorr" {
				continue
			}
			raw, err := json.Marshal(value)
			if err != nil {
				return nil, err
			}
			out[path][k] = raw
		}
	}
	return out, nil
}
func (v *JsonDB) RestoreStorage(data storageData) error {
	for _, path := range []string{"Settings", "Viewed"} {
		filename, _ := v.xPathToFilename(path)
		v.lock(filename)
		err := func() error {
			root, err := v.readJsonFileAsMap(filename)
			if err != nil {
				return err
			}
			if path == "Settings" {
				delete(root, "BitTorr")
			} else {
				root = map[string]interface{}{}
			}
			for k, value := range data[path] {
				root[k] = value
			}
			return v.writeMapAsJsonFile(filename, root)
		}()
		v.unlock(filename)
		if err != nil {
			return err
		}
	}
	return nil
}
func validateStorageJournal(j storageJournal) error {
	if j.Schema != 1 {
		return errors.New("unsupported storage recovery schema")
	}
	for _, data := range []storageData{j.Bolt, j.JSON} {
		if len(data) != 2 || data["Settings"] == nil || data["Viewed"] == nil {
			return errors.New("invalid storage recovery namespaces")
		}
		for path, rows := range data {
			for key, value := range rows {
				if (path == "Settings" && key != "BitTorr") || key == "" || !json.Valid(value) {
					return errors.New("invalid storage recovery entry")
				}
			}
		}
	}
	return nil
}
func recoverStorageTransaction(b, j TorrServerDB) error {
	info, err := os.Lstat(storageJournalPath())
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if ReadOnly {
		return errors.New("storage recovery requires a writable start")
	}
	if !info.Mode().IsRegular() || info.Size() > BackupLimit {
		return errors.New("invalid storage recovery file")
	}
	file, err := os.Open(storageJournalPath())
	if err != nil {
		return err
	}
	data, err := io.ReadAll(io.LimitReader(file, BackupLimit+1))
	_ = file.Close()
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(data) > BackupLimit {
		return errors.New("storage recovery exceeds limit")
	}
	var saved storageJournal
	if json.Unmarshal(data, &saved) != nil {
		return errors.New("invalid storage recovery document")
	}
	if err = validateStorageJournal(saved); err != nil {
		return err
	}
	bb, bok := b.(storageBackend)
	jj, jok := j.(storageBackend)
	if !bok || !jok {
		return errors.New("storage recovery backend unavailable")
	}
	// Leave the journal in place unless BOTH originals are durable again.
	if err = errors.Join(bb.RestoreStorage(saved.Bolt), jj.RestoreStorage(saved.JSON)); err != nil {
		return err
	}
	return os.Remove(storageJournalPath())
}

func SetStoragePreferences(prefs map[string]interface{}) error {
	dbMigrationLock.Lock()
	defer dbMigrationLock.Unlock()
	if ReadOnly || BTsets() == nil {
		return errors.New("storage is read-only or unavailable")
	}
	next := CloneSettings(BTsets())
	for key, value := range prefs {
		if key == "viewedCount" {
			continue
		}
		text, ok := value.(string)
		if !ok || (text != "json" && text != "bbolt") || (key != "settings" && key != "viewed") {
			return errors.New("invalid storage preference")
		}
		if key == "settings" {
			next.StoreSettingsInJson = text == "json"
		} else {
			next.StoreViewedInJson = text == "json"
		}
	}
	old := BTsets()
	if old.StoreSettingsInJson == next.StoreSettingsInJson && old.StoreViewedInJson == next.StoreViewedInJson {
		return nil
	}
	b, j := NewTDB(), NewJsonDB()
	bb, bok := b.(storageBackend)
	jj, jok := j.(storageBackend)
	if !bok || !jok {
		return errors.New("storage backend unavailable")
	}
	if _, err := os.Lstat(storageJournalPath()); !os.IsNotExist(err) {
		return errors.New("resolve the previous storage recovery before migrating")
	}
	saved := storageJournal{Schema: 1}
	var err error
	if saved.Bolt, err = bb.StorageSnapshot(); err != nil {
		return err
	}
	if saved.JSON, err = jj.StorageSnapshot(); err != nil {
		return err
	}
	data, err := json.Marshal(saved)
	if err != nil {
		return err
	}
	if len(data) > BackupLimit {
		return errors.New("storage migration exceeds recovery limit")
	}
	if err = diagnostics.AtomicPrivateFile(storageJournalPath(), data); err != nil {
		return err
	}
	// Stage fresh maps, retaining the recovery snapshot unchanged.
	stagedData, _ := json.Marshal(saved)
	var staged storageJournal
	_ = json.Unmarshal(stagedData, &staged)
	sourceSettings := saved.Bolt["Settings"]
	if old.StoreSettingsInJson {
		sourceSettings = saved.JSON["Settings"]
	}
	sourceViewed := saved.Bolt["Viewed"]
	if old.StoreViewedInJson {
		sourceViewed = saved.JSON["Viewed"]
	}
	raw, _ := json.Marshal(next)
	staged.Bolt["Settings"], staged.JSON["Settings"] = map[string]json.RawMessage{}, map[string]json.RawMessage{}
	targetSettings := staged.Bolt["Settings"]
	if next.StoreSettingsInJson {
		targetSettings = staged.JSON["Settings"]
	}
	for k, v := range sourceSettings {
		targetSettings[k] = v
	}
	targetSettings["BitTorr"] = raw
	staged.Bolt["Viewed"], staged.JSON["Viewed"] = map[string]json.RawMessage{}, map[string]json.RawMessage{}
	if next.StoreViewedInJson {
		staged.JSON["Viewed"] = sourceViewed
	} else {
		staged.Bolt["Viewed"] = sourceViewed
	}
	fail := func(cause error) error {
		rollback := recoverStorageTransaction(b, j)
		if rollback != nil {
			return fmt.Errorf("storage migration failed; recovery journal retained: %w", errors.Join(cause, rollback))
		}
		return fmt.Errorf("storage migration rolled back: %w", cause)
	}
	if err = bb.RestoreStorage(staged.Bolt); err != nil {
		return fail(err)
	}
	if err = jj.RestoreStorage(staged.JSON); err != nil {
		return fail(err)
	}
	actualB, e1 := bb.StorageSnapshot()
	actualJ, e2 := jj.StorageSnapshot()
	expected, _ := json.Marshal(staged)
	actual, _ := json.Marshal(storageJournal{1, actualB, actualJ})
	equal, verify := isByteArraysEqualJson(expected, actual)
	if e1 != nil || e2 != nil || verify != nil || !equal {
		return fail(errors.New("storage verification failed"))
	}
	if err = os.Remove(storageJournalPath()); err != nil {
		return fail(err)
	}
	// Every settings/viewed caller is fenced by dbMigrationLock. Publishing the
	// new router now also prevents writes from landing in the retired backend.
	publishStorageRouting(b, j, next.StoreSettingsInJson, next.StoreViewedInJson)
	StoreBTsets(next)
	if err = saveKnownGood(next); err != nil {
		recordRecovery("settings", "RECOVERY_SNAPSHOT_WRITE_FAILED")
	}
	return nil
}

func publishStorageRouting(b, j TorrServerDB, settingsJSON, viewedJSON bool) {
	cache, ok := tdb.(*DBReadCache)
	if !ok {
		setupDatabaseRouting(b, j, settingsJSON, viewedJSON)
		return
	}
	router, ok := cache.db.(*XPathDBRouter)
	if !ok {
		return
	}
	router.routeMu.Lock()
	router.route2db["settings"] = b
	if settingsJSON {
		router.route2db["settings"] = j
	}
	router.route2db["viewed"] = b
	if viewedJSON {
		router.route2db["viewed"] = j
	}
	router.routeMu.Unlock()
	cache.dataCacheMutex.Lock()
	for key := range cache.dataCache {
		if key[0] == "Settings" || key[0] == "Viewed" {
			delete(cache.dataCache, key)
		}
	}
	cache.dataCacheMutex.Unlock()
	cache.listCacheMutex.Lock()
	delete(cache.listCache, "Settings")
	delete(cache.listCache, "Viewed")
	cache.listCacheMutex.Unlock()
}

// A failed rollback must be repaired before unrelated writes can invalidate its
// durable recovery snapshot. Startup replays it before selecting storage.
func StorageRecoveryPending() bool {
	_, err := os.Lstat(storageJournalPath())
	return !os.IsNotExist(err)
}
