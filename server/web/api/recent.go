package api

import (
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
	config "server/settings"
	"server/torr"
)

func recent(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	rows, err := config.ListRecent()
	if err != nil {
		abortWithErrorCode(c, 500, "RECENT_HISTORY_UNAVAILABLE", err)
		return
	}
	items := []gin.H{}
	for _, row := range rows {
		torrent := torr.GetTorrentInfo(row.Hash)
		if torrent == nil {
			continue
		}
		file := torrent.SubtitleVideoPath(row.FileIndex)
		if file == "" {
			continue
		}
		item := gin.H{"hash": row.Hash, "file_index": row.FileIndex, "last_opened_at": row.LastOpenedAt, "file_path": file, "title": torrent.LibrarySummary().Title}
		for _, viewed := range config.ListViewed(row.Hash) {
			if viewed.FileIndex == row.FileIndex {
				item["viewed"] = viewed
				break
			}
		}
		items = append(items, item)
	}
	c.JSON(200, gin.H{"schema_version": 1, "enabled": config.CurrentFlow().RecentPlaybackHistory, "read_only": config.ReadOnly, "items": items})
}
func recentControl(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4096)
	var request struct {
		Action string `json:"action"`
		Hash   string `json:"hash"`
		Index  int    `json:"file_index"`
	}
	if c.ShouldBindJSON(&request) != nil || request.Action != "clear" && request.Action != "remove" || request.Action == "remove" && (request.Hash == "" || request.Index < 1) {
		abortWithErrorCode(c, 400, "INVALID_RECENT_ACTION", errors.New("invalid history action"))
		return
	}
	if config.ReadOnly {
		abortWithErrorCode(c, 403, "READ_ONLY", errors.New("database is read-only"))
		return
	}
	if err := config.RemoveRecent(request.Hash, request.Index, request.Action == "clear"); err != nil {
		abortWithErrorCode(c, 500, "RECENT_HISTORY_UNAVAILABLE", err)
		return
	}
	c.JSON(200, gin.H{"schema_version": 1, "ok": true})
}
