package settings

import (
	"encoding/json"
	"time"

	"server/log"
)

// Viewed marks one torrent file as watched. TimeCode is the playback position
// in seconds; it is stored only while BTSets.TrackTimecode is on (otherwise 0).
type Viewed struct {
	PositionSource    string     `json:"position_source,omitempty"`
	PositionUpdatedAt *time.Time `json:"position_updated_at,omitempty"`
	Completed         *bool      `json:"completed,omitempty"`
	Hash              string     `json:"hash"`
	FileIndex         int        `json:"file_index"`
	TimeCode          float64    `json:"timecode"`
}

// readIndexes decodes a torrent's viewed record: file index -> timecode. It
// also accepts the pre-timecode format ({"1":{}}), reading those marks as 0.
func readIndexes(buf []byte) map[int]float64 {
	m := map[int]float64{}
	if json.Unmarshal(buf, &m) == nil {
		return m
	}
	var old map[int]struct{}
	if json.Unmarshal(buf, &old) == nil {
		for k := range old {
			m[k] = 0
		}
	}
	return m
}

func trackTimecode() bool {
	sets := BTsets()
	return sets != nil && sets.TrackTimecode
}

// SetViewed stores the mark with the caller's timecode (a client reporting
// its position through /viewed).
func SetViewed(vv *Viewed) {
	dbMigrationLock.RLock()
	defer dbMigrationLock.RUnlock()
	val := vv.TimeCode
	m := readIndexes(tdb.Get("Viewed", vv.Hash))
	// Explicit completion edits do not manufacture or erase a saved position.
	if vv.Completed != nil && vv.PositionSource == "" && val == 0 {
		val = m[vv.FileIndex]
	}
	updateViewedMetadataLocked(vv, trackTimecode() && (vv.TimeCode > 0 || vv.PositionSource != ""))
	if !trackTimecode() && !(vv.Completed != nil && vv.PositionSource == "" && vv.TimeCode == 0) {
		val = 0
	}

	m[vv.FileIndex] = val
	buf, err := json.Marshal(m)
	if err == nil {
		tdb.Set("Viewed", vv.Hash, buf)
	} else {
		log.TLogln("Error set viewed:", err)
	}
}

// MarkViewed marks a file as watched without touching a timecode already
// stored for it. Internal callers (playback start, the bot) know nothing about
// the position, and SetViewed with 0 would reset the one a client saved.
func MarkViewed(hash string, fileIndex int) {
	dbMigrationLock.RLock()
	defer dbMigrationLock.RUnlock()
	m := readIndexes(tdb.Get("Viewed", hash))
	if _, ok := m[fileIndex]; ok {
		return
	}
	m[fileIndex] = 0
	buf, err := json.Marshal(m)
	if err == nil {
		tdb.Set("Viewed", hash, buf)
	} else {
		log.TLogln("Error set viewed:", err)
	}
}

func RemViewed(vv *Viewed) {
	dbMigrationLock.RLock()
	defer dbMigrationLock.RUnlock()
	removeViewedMetadataLocked(vv.Hash, vv.FileIndex)
	if vv.FileIndex == -1 {
		tdb.Rem("Viewed", vv.Hash)
		return
	}
	buf := tdb.Get("Viewed", vv.Hash)
	if len(buf) == 0 {
		return
	}
	m := readIndexes(buf)
	delete(m, vv.FileIndex)
	buf, err := json.Marshal(m)
	if err != nil {
		log.TLogln("Error rem viewed:", err)
		return
	}
	tdb.Set("Viewed", vv.Hash, buf)
}

func ListViewed(hash string) []*Viewed {
	dbMigrationLock.RLock()
	defer dbMigrationLock.RUnlock()
	recentMu.Lock()
	metadata := viewedMetadataLocked()
	recentMu.Unlock()
	makeViewed := func(hash string, index int, tc float64) *Viewed {
		v := &Viewed{Hash: hash, FileIndex: index, TimeCode: tc}
		for _, row := range metadata.Items {
			if row.Hash == hash && row.FileIndex == index {
				v.PositionSource = row.PositionSource
				v.PositionUpdatedAt = row.PositionUpdatedAt
				v.Completed = row.Completed
				break
			}
		}
		if tc > 0 && v.PositionSource == "" {
			v.PositionSource = "legacy_saved"
		}
		return v
	}
	if hash != "" {
		buf := tdb.Get("Viewed", hash)
		if len(buf) == 0 {
			return []*Viewed{}
		}
		var ret []*Viewed
		for i, tc := range readIndexes(buf) {
			ret = append(ret, makeViewed(hash, i, tc))
		}
		return ret
	}

	var ret []*Viewed
	for _, key := range tdb.List("Viewed") {
		buf := tdb.Get("Viewed", key)
		if len(buf) == 0 {
			continue
		}
		for i, tc := range readIndexes(buf) {
			ret = append(ret, makeViewed(key, i, tc))
		}
	}
	return ret
}
