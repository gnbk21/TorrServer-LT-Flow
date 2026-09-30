package settings

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	bolt "go.etcd.io/bbolt"
)

func backupTestDB(t testing.TB) *TDB {
	t.Helper()
	oldDB, oldPath, oldSettings, oldReadOnly := tdb, Path, BTsets(), ReadOnly
	Path = t.TempDir()
	ReadOnly = false
	database, err := bolt.Open(filepath.Join(Path, "config.db"), 0600, nil)
	if err != nil {
		t.Fatal(err)
	}
	db := &TDB{Path: Path, db: database}
	router := NewXPathDBRouter()
	router.RegisterRoute(db, "")
	tdb = NewDBReadCache(router)
	StoreBTsets(&BTSets{CacheSize: 64 << 20, ReaderReadAHead: 95, PreloadCache: 50, Flow: DefaultFlowSettings(), JacRedKey: "private-secret", JacRedUrl: "https://example.test/passkey"})
	t.Cleanup(func() {
		database.Close()
		tdb = oldDB
		Path = oldPath
		StoreBTsets(oldSettings)
		ReadOnly = oldReadOnly
	})
	return db
}

func TestBackupRejectsNestedUnknownNullAndInvalidBounds(t *testing.T) {
	backupTestDB(t)
	for _, field := range []struct{ name, value string }{
		{"Flow", `{"SwarmCustom":{"PrivateKey":"secret"}}`},
		{"Flow", `{"UnknownField":1}`}, {"CacheSize", `null`},
		{"TorrentDisconnectTimeout", `-1`}, {"RetrackersMode", `9`},
	} {
		backup, _ := ExportBackup()
		backup.Settings[field.name] = json.RawMessage(field.value)
		data, _ := json.Marshal(backup)
		if _, err := ParseBackup(data); err == nil {
			t.Fatalf("accepted %s=%s", field.name, field.value)
		}
	}
}

func BenchmarkLibrarySingleUpsert(b *testing.B) {
	for _, legacy := range []bool{true, false} {
		name := "single-key"
		if legacy {
			name = "legacy-whole-library"
		}
		b.Run(name, func(b *testing.B) {
			backupTestDB(b)
			for i := 0; i < 1000; i++ {
				AddTorrent(&TorrentDB{TorrentSpec: &TorrentSpec{InfoHash: fmt.Sprintf("%040x", i)}, Title: "Generated library fixture"})
			}
			row := &TorrentDB{TorrentSpec: &TorrentSpec{InfoHash: fmt.Sprintf("%040x", 500)}, Title: "Edited title"}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if legacy {
					rows := ListTorrent()
					for _, existing := range rows {
						if existing.InfoHash == row.InfoHash {
							existing.Title = row.Title
						}
						value, _ := json.Marshal(existing)
						tdb.Set("Torrents", existing.InfoHash, value)
					}
				} else {
					AddTorrent(row)
				}
			}
		})
	}
}

func TestPortableBackupExcludesCredentialsAndPreservesLocalSecrets(t *testing.T) {
	backupTestDB(t)
	AddTorrent(&TorrentDB{TorrentSpec: &TorrentSpec{InfoHash: strings.Repeat("a", 40), InfoBytes: []byte("private-passkey"), Trackers: [][]string{{"https://private.test/passkey"}}}, Title: "Episode", Data: "secret-client-data", Poster: "https://host/passkey"})
	backup, err := ExportBackup()
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(backup)
	for _, secret := range []string{"private-secret", "passkey", "secret-client-data", "InfoBytes", "Poster"} {
		if strings.Contains(string(data), secret) {
			t.Fatal("secret exported", secret)
		}
	}
	parsed, err := ParseBackup(data)
	if err != nil {
		t.Fatal(err)
	}
	merged, err := parsed.MergedSettings()
	if err != nil || merged.JacRedKey != "private-secret" || merged.JacRedUrl != "https://example.test/passkey" {
		t.Fatal("host secrets not preserved", err)
	}
	if parsed.Digest() != backup.Digest() {
		t.Fatal("preview digest mismatch")
	}
	backup.Settings["JacRedKey"] = json.RawMessage(`"injected"`)
	data, _ = json.Marshal(backup)
	if _, err = ParseBackup(data); err == nil {
		t.Fatal("import accepted secret setting")
	}
}

func TestLibraryRestoreIsTransactionalAndInvalidatesCache(t *testing.T) {
	backupTestDB(t)
	hash := strings.Repeat("b", 40)
	AddTorrent(&TorrentDB{TorrentSpec: &TorrentSpec{InfoHash: hash}, Title: "Original"})
	if GetTorrent(hash).Title != "Original" {
		t.Fatal("initial record")
	}
	good := &TorrentDB{TorrentSpec: &TorrentSpec{InfoHash: hash}, Title: "Updated"}
	if err := WriteLibraryBatch([]*TorrentDB{good, {TorrentSpec: &TorrentSpec{InfoHash: "bad"}}}, false); err == nil {
		t.Fatal("invalid batch succeeded")
	}
	if GetTorrent(hash).Title != "Original" {
		t.Fatal("partial batch escaped transaction")
	}
	if err := WriteLibraryBatch([]*TorrentDB{good}, false); err != nil {
		t.Fatal(err)
	}
	if GetTorrent(hash).Title != "Updated" {
		t.Fatal("stale cached record")
	}
}

func TestFailedSettingsWriteDoesNotPublishCachedSuccess(t *testing.T) {
	db := backupTestDB(t)
	if err := SetBTSetsChecked(BTsets()); err != nil {
		t.Fatal(err)
	}
	before := BTsets()
	next := *before
	next.CacheSize = 128 << 20
	db.db.Close()
	if err := SetBTSetsChecked(&next); err == nil {
		t.Fatal("closed database accepted write")
	}
	if BTsets().CacheSize != 64<<20 {
		t.Fatal("failed settings were published")
	}
	var cached BTSets
	if err := json.Unmarshal(tdb.Get("Settings", "BitTorr"), &cached); err != nil || cached.CacheSize != 64<<20 {
		t.Fatal("cache claimed failed write succeeded")
	}
}

func TestCorruptJSONDatabaseCannotBeSilentlyOverwritten(t *testing.T) {
	directory := t.TempDir()
	file := filepath.Join(directory, "settings.json")
	if err := os.WriteFile(file, []byte(`{"broken"`), 0600); err != nil {
		t.Fatal(err)
	}
	database := &JsonDB{Path: directory, xPathDelimeter: "/", filenameDelimiter: ".", filenameExtension: ".json", fileMode: 0600}
	if err := database.PutChecked("Settings", "BitTorr", []byte(`{"CacheSize":123}`)); err == nil {
		t.Fatal("corrupt database overwritten")
	}
	contents, _ := os.ReadFile(file)
	if string(contents) != `{"broken"` {
		t.Fatal("original data lost")
	}
}

func FuzzBackupImport(f *testing.F) {
	f.Add([]byte(`{"schema_version":1,"kind":"TorrServer-Flow portable backup","created_at":"2026-09-30T00:00:00Z","settings":{},"library":[]}`))
	f.Add([]byte(`null`))
	f.Add([]byte(`{"settings":{"__proto__":{}}}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 65536 {
			return
		}
		backup, err := ParseBackup(data)
		if err != nil {
			return
		}
		if backup.SchemaVersion != 1 || len(backup.Library) > BackupTorrentLimit {
			t.Fatal("invalid import accepted")
		}
		canonical, _ := json.Marshal(backup)
		if _, err := ParseBackup(canonical); err != nil {
			t.Fatal("accepted import is not stable", err)
		}
	})
}

func FuzzFlowSettings(f *testing.F) {
	f.Add([]byte(`{"Enabled":false,"StartupBufferMinMB":-1,"SwarmProfile":"CUSTOM"}`))
	f.Add([]byte(`null`))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 65536 {
			return
		}
		var value FlowSettings
		if json.Unmarshal(data, &value) != nil {
			return
		}
		value.Normalize()
		if value.StartupBufferMinMB < 1 || value.StartupBufferMaxMB < value.StartupBufferMinMB || value.MaxBufferSeconds < value.TargetBufferSeconds || value.NetworkRetryMaxSec < value.NetworkRetryMinSec {
			t.Fatal("unsafe normalized settings")
		}
	})
}
