package settings

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"time"

	bolt "go.etcd.io/bbolt"
)

const BackupLimit = 16 << 20
const BackupTorrentLimit = 2000

type Backup struct {
	SchemaVersion int                        `json:"schema_version"`
	Kind          string                     `json:"kind"`
	CreatedAt     time.Time                  `json:"created_at"`
	Settings      map[string]json.RawMessage `json:"settings"`
	Library       []BackupTorrent            `json:"library"`
}
type BackupTorrent struct {
	Hash        string `json:"hash"`
	Title       string `json:"title"`
	Category    string `json:"category"`
	DisplayName string `json:"display_name"`
	Timestamp   int64  `json:"timestamp"`
	Size        int64  `json:"size"`
}

// Portable backups exclude accounts, keys, private tracker URLs, arbitrary
// client Data, poster URLs, filesystem paths and storage-routing preferences.
// Restoring merges the portable fields into the host's current configuration.
var portableSettingNames = []string{
	"Flow", "CacheSize", "ReaderReadAHead", "PreloadCache", "PadTailPartial",
	"RemoveCacheOnDrop", "ForceEncrypt", "RetrackersMode", "TorrentDisconnectTimeout",
	"EnableDebug", "EnableDLNA", "FriendlyName", "EnableBonjour", "EnableRutorSearch",
	"EnableTorznabSearch", "EnableJacRedSearch", "EnableIPv6", "DisableTCP", "DisableUTP",
	"DisableUPNP", "DisableDHT", "DisablePEX", "DisableUpload", "DisableEndGame",
	"DownloadRateLimit", "UploadRateLimit", "ConnectionsLimit", "DHTConnectionsLimit",
	"PeersListenPort", "EnableLPD", "ShowFSActiveTorr", "TrackTimecode", "MergeAllM3U",
}
var backupHash = regexp.MustCompile(`^[a-f0-9]{40}$`)

func ExportBackup() (*Backup, error) {
	if BTsets() == nil || tdb == nil {
		return nil, errors.New("settings are not ready")
	}
	raw, err := json.Marshal(BTsets())
	if err != nil {
		return nil, err
	}
	var fields map[string]json.RawMessage
	if err = json.Unmarshal(raw, &fields); err != nil {
		return nil, err
	}
	out := &Backup{SchemaVersion: 1, Kind: "TorrServer-Flow portable backup", CreatedAt: time.Now().UTC(), Settings: map[string]json.RawMessage{}, Library: []BackupTorrent{}}
	for _, name := range portableSettingNames {
		out.Settings[name] = fields[name]
	}
	rows := ListTorrent()
	if len(rows) > BackupTorrentLimit {
		return nil, errors.New("library exceeds portable backup limit")
	}
	for _, row := range rows {
		if row != nil && row.TorrentSpec != nil {
			out.Library = append(out.Library, BackupTorrent{row.InfoHash, row.Title, row.Category, row.DisplayName, row.Timestamp, row.Size})
		}
	}
	return out, nil
}

func ParseBackup(data []byte) (*Backup, error) {
	if len(data) > BackupLimit {
		return nil, errors.New("backup exceeds 16 MiB")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var backup Backup
	if err := decoder.Decode(&backup); err != nil {
		return nil, errors.New("invalid backup JSON")
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, errors.New("backup has trailing JSON")
	}
	if backup.SchemaVersion != 1 || backup.Kind != "TorrServer-Flow portable backup" || backup.CreatedAt.IsZero() || backup.Settings == nil || backup.Library == nil || len(backup.Library) > BackupTorrentLimit {
		return nil, errors.New("unsupported backup schema or library limit")
	}
	allowed := map[string]bool{}
	for _, key := range portableSettingNames {
		allowed[key] = true
	}
	for key := range backup.Settings {
		if !allowed[key] {
			return nil, fmt.Errorf("nonportable settings field: %s", key)
		}
	}
	// Decode and normalize a deep copy to check types and bounds before preview.
	if _, err := backup.MergedSettings(); err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for _, row := range backup.Library {
		if !backupHash.MatchString(row.Hash) || seen[row.Hash] || row.Size < 0 || row.Timestamp < 0 || len(row.Title) > 4096 || len(row.DisplayName) > 4096 || len(row.Category) > 512 {
			return nil, errors.New("invalid or duplicate library record")
		}
		seen[row.Hash] = true
	}
	return &backup, nil
}

func (b *Backup) Digest() string {
	data, _ := json.Marshal(b)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
func (b *Backup) MergedSettings() (*BTSets, error) {
	for name, value := range b.Settings {
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return nil, errors.New("backup settings cannot be null")
		}
		if name == "Flow" {
			decoder := json.NewDecoder(bytes.NewReader(value))
			decoder.DisallowUnknownFields()
			type portableFlow FlowSettings // bypass the migration decoder to reject unknown fields
			var checked portableFlow
			if err := decoder.Decode(&checked); err != nil {
				return nil, errors.New("invalid or unknown Flow settings")
			}
		}
	}
	current := BTsets()
	if current == nil {
		current = &BTSets{Flow: DefaultFlowSettings()}
	}
	raw, _ := json.Marshal(current)
	var fields map[string]json.RawMessage
	json.Unmarshal(raw, &fields)
	for name, value := range b.Settings {
		fields[name] = value
	}
	raw, err := json.Marshal(fields)
	if err != nil {
		return nil, err
	}
	var sets BTSets
	if err = json.Unmarshal(raw, &sets); err != nil {
		return nil, errors.New("backup settings have invalid types")
	}
	if sets.CacheSize < 0 || sets.CacheSize > 16<<30 || sets.ConnectionsLimit < 0 || sets.ConnectionsLimit > 10000 || sets.DHTConnectionsLimit < 0 || sets.DHTConnectionsLimit > 100000 || sets.ReaderReadAHead < 0 || sets.ReaderReadAHead > 100 || sets.PreloadCache < 0 || sets.PreloadCache > 100 || sets.PeersListenPort < 0 || sets.PeersListenPort > 65535 || sets.DownloadRateLimit < 0 || sets.UploadRateLimit < 0 || sets.TorrentDisconnectTimeout < 0 || sets.TorrentDisconnectTimeout > 86400 || sets.RetrackersMode < 0 || sets.RetrackersMode > 3 || len(sets.FriendlyName) > 512 {
		return nil, errors.New("backup settings exceed supported bounds")
	}
	if sets.Flow != nil {
		before, _ := json.Marshal(sets.Flow)
		sets.Flow.Normalize()
		after, _ := json.Marshal(sets.Flow)
		if !bytes.Equal(before, after) {
			return nil, errors.New("backup Flow settings exceed supported bounds")
		}
	}
	return &sets, nil
}

// Full recovery snapshots are saved locally with owner-only access before
// mutation. They are never exposed through the portable export endpoint.
type RecoveryBackup struct {
	Settings *BTSets      `json:"settings"`
	Library  []*TorrentDB `json:"library"`
}

func CaptureRecovery() RecoveryBackup {
	raw, _ := json.Marshal(BTsets())
	var sets BTSets
	json.Unmarshal(raw, &sets)
	return RecoveryBackup{&sets, ListTorrent()}
}
func (b *Backup) MergedLibrary(old []*TorrentDB) []*TorrentDB {
	records := map[string]*TorrentDB{}
	for _, row := range old {
		if row != nil && row.TorrentSpec != nil {
			records[row.InfoHash] = row
		}
	}
	out := make([]*TorrentDB, 0, len(b.Library))
	for _, row := range b.Library {
		record := records[row.Hash]
		if record == nil {
			record = &TorrentDB{TorrentSpec: &TorrentSpec{InfoHash: row.Hash}}
		}
		copy := *record
		spec := *record.TorrentSpec
		copy.TorrentSpec = &spec
		copy.Title, copy.Category, copy.DisplayName, copy.Timestamp = row.Title, row.Category, row.DisplayName, row.Timestamp
		if row.Size > 0 || copy.Size == 0 {
			copy.Size = row.Size
		}
		out = append(out, &copy)
	}
	return out
}

// Torrents always use bbolt in the existing router. Apply their batch in one
// transaction, then invalidate only the relevant read-cache entries.
func WriteLibraryBatch(rows []*TorrentDB, replace bool) error {
	if ReadOnly {
		return errors.New("database is read-only")
	}
	dbMigrationLock.RLock()
	defer dbMigrationLock.RUnlock()
	mu.Lock()
	defer mu.Unlock()
	cache, ok := tdb.(*DBReadCache)
	if !ok {
		return errors.New("unexpected library cache")
	}
	router, ok := cache.db.(*XPathDBRouter)
	if !ok {
		return errors.New("unexpected library router")
	}
	database, ok := router.getDBForXPath("Torrents").(*TDB)
	if !ok || database.db == nil {
		return errors.New("library database is not available")
	}
	err := database.db.Update(func(tx *bolt.Tx) error {
		if replace {
			if err := tx.DeleteBucket([]byte("Torrents")); err != nil && err != bolt.ErrBucketNotFound {
				return err
			}
		}
		bucket, err := tx.CreateBucketIfNotExists([]byte("Torrents"))
		if err != nil {
			return err
		}
		for _, row := range rows {
			if row == nil || row.TorrentSpec == nil || !backupHash.MatchString(row.InfoHash) {
				return errors.New("invalid recovery record")
			}
			data, err := json.Marshal(row)
			if err != nil {
				return err
			}
			if err = bucket.Put([]byte(row.InfoHash), data); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	cache.dataCacheMutex.Lock()
	for key := range cache.dataCache {
		if key[0] == "Torrents" {
			delete(cache.dataCache, key)
		}
	}
	cache.dataCacheMutex.Unlock()
	cache.listCacheMutex.Lock()
	delete(cache.listCache, "Torrents")
	cache.listCacheMutex.Unlock()
	return nil
}
