package api

import (
	"encoding/json"
	"errors"
	"server/log"
	"server/torr/state"
	"server/torrshash"
	"sort"
	"strings"
)

const libraryExportLimit = 16 << 20
const libraryExportRows = 5000

// An explicit projection prevents newer/legacy DTO fields, arbitrary client
// Data and opaque tracker-bearing tokens from crossing the export boundary.
type libraryExportRow struct {
	Hash      string `json:"hash"`
	Title     string `json:"title"`
	Category  string `json:"category"`
	TorrsHash string `json:"torrs_hash,omitempty"`
}

func projectLibraryExport(rows []state.TorrentStatus) ([]libraryExportRow, error) {
	if len(rows) > libraryExportRows {
		return nil, errors.New("library export exceeds 5000 entries")
	}
	out := make([]libraryExportRow, 0, len(rows))
	for _, row := range rows {
		hash := strings.ToLower(row.Hash)
		if !validLibraryExportHash(hash) || len(row.Title) > 4096 || len(row.Category) > 512 {
			return nil, errors.New("library contains an invalid export record")
		}
		entry := libraryExportRow{Hash: hash, Title: log.RedactSecrets(row.Title), Category: log.RedactSecrets(row.Category)}
		if len(hash) == 40 {
			token := torrshash.New(hash)
			token.AddField(torrshash.TagTitle, entry.Title)
			token.AddField(torrshash.TagCategory, entry.Category)
			var err error
			entry.TorrsHash, err = torrshash.Pack(token)
			if err != nil {
				return nil, errors.New("cannot encode library export")
			}
		}
		out = append(out, entry)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Hash < out[j].Hash })
	data, err := json.Marshal(out)
	if err != nil || len(data) > libraryExportLimit {
		return nil, errors.New("library export exceeds 16 MiB")
	}
	return out, nil
}

func validLibraryExportHash(hash string) bool {
	if len(hash) != 40 && len(hash) != 64 {
		return false
	}
	for _, char := range hash {
		if !(char >= '0' && char <= '9' || char >= 'a' && char <= 'f') {
			return false
		}
	}
	return true
}
