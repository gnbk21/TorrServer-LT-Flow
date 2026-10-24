package settings

import (
	"encoding/json"

	"server/log"
)

// Viewed marks one torrent file as watched. TimeCode is the playback position
// in seconds; it is stored only while BTSets.TrackTimecode is on (otherwise 0).
type Viewed struct {
	Hash      string  `json:"hash"`
	FileIndex int     `json:"file_index"`
	TimeCode  float64 `json:"timecode"`
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
	if !trackTimecode() {
		val = 0
	}

	m := readIndexes(tdb.Get("Viewed", vv.Hash))
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
	if hash != "" {
		buf := tdb.Get("Viewed", hash)
		if len(buf) == 0 {
			return []*Viewed{}
		}
		var ret []*Viewed
		for i, tc := range readIndexes(buf) {
			ret = append(ret, &Viewed{Hash: hash, FileIndex: i, TimeCode: tc})
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
			ret = append(ret, &Viewed{Hash: key, FileIndex: i, TimeCode: tc})
		}
	}
	return ret
}
