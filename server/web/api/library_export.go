package api

import (
	"net/http"
	"server/torr"
	"server/torr/state"

	"github.com/gin-gonic/gin"
)

func flowLibraryExport(c *gin.Context) {
	rows := make([]state.TorrentStatus, 0)
	for _, torrent := range torr.LibraryTorrents() {
		if c.Request.Context().Err() != nil {
			return
		}
		if len(rows) >= libraryExportRows {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "library export exceeds 5000 entries"})
			return
		}
		rows = append(rows, torrent.LibrarySummary())
	}
	export, err := projectLibraryExport(rows)
	if err != nil {
		abortWithJSONError(c, http.StatusRequestEntityTooLarge, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, export)
}
