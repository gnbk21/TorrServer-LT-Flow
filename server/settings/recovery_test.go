package settings

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSettingsRecoverPreserveAndExplicitRepair(t *testing.T) {
	backupTestDB(t)
	oldDB := tdb
	db := &JsonDB{Path: Path, filenameDelimiter: ".", filenameExtension: ".json", fileMode: 0600, xPathDelimeter: "/"}
	tdb = db
	t.Cleanup(func() { tdb = oldDB; recordRecovery("settings", "") })
	saved := CloneSettings(BTsets())
	saved.CacheSize = 2048 << 20
	if err := SetBTSetsChecked(saved); err != nil {
		t.Fatal(err)
	}
	corrupt := []byte(`{"BitTorr":`)
	if err := os.WriteFile(filepath.Join(Path, "settings.json"), corrupt, 0600); err != nil {
		t.Fatal(err)
	}
	StoreBTsets(nil)
	loadBTSets()
	if BTsets().CacheSize != 2048<<20 || SettingsRecovery().Source != "last_known_good" {
		t.Fatal("recovery lost intended cache", SettingsRecovery())
	}
	actual, _ := os.ReadFile(filepath.Join(Path, "settings.json"))
	if !bytes.Equal(actual, corrupt) {
		t.Fatal("load overwrote damaged source")
	}
	if err := SetBTSetsChecked(CloneSettings(BTsets())); err != nil {
		t.Fatal("explicit repair failed", err)
	}
	backups, _ := filepath.Glob(filepath.Join(Path, "flow-corrupt-settings-*.json"))
	if len(backups) != 1 {
		t.Fatal("original was not retained")
	}
	actual, _ = os.ReadFile(backups[0])
	if !bytes.Equal(actual, corrupt) {
		t.Fatal("original changed")
	}
	actual, _ = os.ReadFile(filepath.Join(Path, "settings.json"))
	if !json.Valid(actual) {
		t.Fatal("repair is invalid")
	}
}

func TestRejectedSchemaSurvivesExplicitRepair(t *testing.T) {
	for _, backend := range []string{"bbolt", "json"} {
		t.Run(backend, func(t *testing.T) {
			backupTestDB(t)
			if backend == "json" {
				tdb = &JsonDB{Path: Path, filenameDelimiter: ".", filenameExtension: ".json", fileMode: 0600, xPathDelimeter: "/"}
			}
			t.Cleanup(func() { recordRecovery("settings", "") })
			good := CloneSettings(BTsets())
			if err := SetBTSetsChecked(good); err != nil {
				t.Fatal(err)
			}
			future := CloneSettings(good)
			future.Flow.SchemaVersion = 2
			rejected, _ := json.Marshal(future)
			if err := putChecked(tdb, "Settings", "BitTorr", rejected); err != nil {
				t.Fatal(err)
			}
			// The JSON backend normalizes object key order when serializing.
			rejected = append([]byte(nil), tdb.Get("Settings", "BitTorr")...)
			loadBTSets()
			if SettingsRecovery().Source != "last_known_good" || !bytes.Equal(tdb.Get("Settings", "BitTorr"), rejected) {
				t.Fatal("load replaced rejected schema", SettingsRecovery())
			}
			if err := SetBTSetsChecked(CloneSettings(BTsets())); err != nil {
				t.Fatal(err)
			}
			backups, _ := filepath.Glob(filepath.Join(Path, "flow-rejected-settings-*.json"))
			if len(backups) != 1 {
				t.Fatal("rejected schema was not retained")
			}
			original, _ := os.ReadFile(backups[0])
			if !bytes.Equal(original, rejected) {
				t.Fatal("rejected schema changed")
			}
		})
	}
}

func TestRecoveryFailureDoesNotPublishSettings(t *testing.T) {
	backupTestDB(t)
	before := CloneSettings(BTsets())
	next := CloneSettings(before)
	next.CacheSize *= 2
	if err := os.Mkdir(filepath.Join(Path, "flow-settings-good.json"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := SetBTSetsChecked(next); err == nil {
		t.Fatal("snapshot storage failure ignored")
	}
	if SettingsRevision(before) != SettingsRevision(BTsets()) {
		t.Fatal("failed write became effective")
	}
}
func TestSettingsHotApplicationAndSchema(t *testing.T) {
	old := NewDefaultConfig()
	next := CloneSettings(old)
	next.CacheSize = 2048 << 20
	next.Flow.TargetBufferSeconds = 90
	next.Flow.WarmCacheBudgetMB = 0
	if NeedsEngineRestart(old, next) {
		t.Fatal("safe cache policy restarted engine")
	}
	next.DisableDHT = !old.DisableDHT
	if !NeedsEngineRestart(old, next) {
		t.Fatal("native change classified hot")
	}
	next = CloneSettings(old)
	next.UseDisk = true
	next.TorrentsSavePath = t.TempDir()
	if !NeedsEngineRestart(old, next) {
		t.Fatal("disk ownership classified hot")
	}
	next.Flow.SchemaVersion = 2
	if ValidateSettings(next) == nil {
		t.Fatal("future schema accepted")
	}
}

func TestSettingsRejectQuotedActiveDiskCache(t *testing.T) {
	backupTestDB(t)
	before := SettingsRevision(BTsets())
	s := NewDefaultConfig()
	s.UseDisk = true
	s.TorrentsSavePath = `"C:\cache folder"`
	if ValidateSettings(s) == nil {
		t.Fatal("settings accepted a quoted disk cache path")
	}
	if err := SetBTSetsChecked(s); err == nil {
		t.Fatal("settings persisted a quoted disk cache path")
	}
	if SettingsRevision(BTsets()) != before {
		t.Fatal("rejected disk path changed the active settings")
	}
	s.TorrentsSavePath = `C:\cache folder`
	if err := ValidateSettings(s); err != nil {
		t.Fatal("settings rejected an unquoted disk cache path", err)
	}
}

func TestRecoveryRejectsTrailingDataAndPortablePolicyPreserved(t *testing.T) {
	backupTestDB(t)
	s := CloneSettings(BTsets())
	s.Flow.ManagementOrigins = "https://local-controller.example"
	s.Flow.RequirePlaybackToken = true
	s.Flow.TorrentInterface = "local-adapter"
	StoreBTsets(s)
	backup, err := ExportBackup()
	if err != nil {
		t.Fatal(err)
	}
	var portable map[string]any
	if err := json.Unmarshal(backup.Settings["Flow"], &portable); err != nil {
		t.Fatal(err)
	}
	for _, name := range hostFlowFields {
		if _, ok := portable[name]; ok {
			t.Fatal("host policy exported", name)
		}
	}
	merged, err := backup.MergedSettings()
	if err != nil {
		t.Fatal(err)
	}
	if merged.Flow.TorrentInterface != s.Flow.TorrentInterface || !merged.Flow.RequirePlaybackToken || merged.Flow.ManagementOrigins != s.Flow.ManagementOrigins {
		t.Fatal("restore replaced host policy")
	}
	if err := saveKnownGood(s); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(Path, "flow-settings-good.json")
	b, _ := os.ReadFile(path)
	if err := os.WriteFile(path, append(b, []byte(` {}`)...), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readKnownGood(); err == nil {
		t.Fatal("trailing recovery data accepted")
	}
}
