package settings

import (
	"encoding/json"
	"sort"
	"sync"
)

// TorrentSpec is the persistence-layer subset of the engine spec. Field
// names and JSON shape are kept compatible with the anacrolix-era
// metainfo encoding so pre-existing config.db files load unchanged.
type TorrentSpec struct {
	InfoHash    string     `json:"InfoHash"` // 40-hex lowercase
	InfoBytes   []byte     `json:"InfoBytes,omitempty"`
	Trackers    [][]string `json:"Trackers,omitempty"`
	DisplayName string     `json:"DisplayName,omitempty"`
}

// TorrentDB is the row persisted in the `Torrents` bucket.
type TorrentDB struct {
	*TorrentSpec

	Title    string `json:"title,omitempty"`
	Category string `json:"category,omitempty"`
	Poster   string `json:"poster,omitempty"`
	Data     string `json:"data,omitempty"`

	Timestamp int64 `json:"timestamp,omitempty"`
	Size      int64 `json:"size,omitempty"`
}

// File is kept for callers that historically marshalled it inside Data.
type File struct {
	Name string `json:"name,omitempty"`
	Id   int    `json:"id,omitempty"`
	Size int64  `json:"size,omitempty"`
}

var mu sync.Mutex

// AddTorrent upserts a torrent record by info hash.
func AddTorrent(torr *TorrentDB) {
	if torr == nil || torr.TorrentSpec == nil {
		return
	}
	dbMigrationLock.RLock()
	defer dbMigrationLock.RUnlock()
	mu.Lock()
	defer mu.Unlock()
	if buf, err := json.Marshal(torr); err == nil {
		tdb.Set("Torrents", torr.InfoHash, buf)
	}
}

// ListTorrent returns every persisted torrent record.
func ListTorrent() []*TorrentDB {
	dbMigrationLock.RLock()
	defer dbMigrationLock.RUnlock()
	mu.Lock()
	defer mu.Unlock()

	var list []*TorrentDB
	for _, key := range tdb.List("Torrents") {
		buf := tdb.Get("Torrents", key)
		if len(buf) == 0 {
			continue
		}
		var t *TorrentDB
		if err := json.Unmarshal(buf, &t); err != nil || t == nil {
			continue
		}
		list = append(list, t)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Timestamp > list[j].Timestamp })
	return list
}

// GetTorrent returns a single torrent record by info-hash hex, or nil. It is a
// direct key lookup (records are stored under their InfoHash, same key RemTorrent
// uses) — unlike ListTorrent it does not read and unmarshal the whole bucket, so
// it is cheap enough for frequent status polling.
func GetTorrent(hashHex string) *TorrentDB {
	dbMigrationLock.RLock()
	defer dbMigrationLock.RUnlock()
	mu.Lock()
	defer mu.Unlock()

	buf := tdb.Get("Torrents", hashHex)
	if len(buf) == 0 {
		return nil
	}
	var t *TorrentDB
	if err := json.Unmarshal(buf, &t); err != nil {
		return nil
	}
	return t
}

// RemTorrent removes a torrent record by 40-hex info hash.
func RemTorrent(hashHex string) {
	mu.Lock()
	defer mu.Unlock()
	tdb.Rem("Torrents", hashHex)
}
