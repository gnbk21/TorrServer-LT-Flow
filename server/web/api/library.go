package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"server/torr"
	"server/torr/state"
	"strconv"
	"time"
)

func flowLibrary(c *gin.Context) {
	page, limit := 1, 50
	for key, target := range map[string]*int{"page": &page, "limit": &limit} {
		if raw, exists := c.GetQuery(key); exists {
			value, err := strconv.Atoi(raw)
			if err != nil || value < 1 || (key == "limit" && value > 100) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid pagination"})
				return
			}
			*target = value
		}
	}
	query, category, order := c.Query("q"), c.Query("category"), c.DefaultQuery("sort", "recent")
	if len(query) > 1024 || len(category) > 1024 || (category != "" && category != "uncategorized" && (len(category) < 9 || category[:9] != "category:")) || (order != "recent" && order != "title" && order != "size") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid library filter"})
		return
	}
	rows := []state.TorrentStatus{}
	for _, torrent := range torr.LibraryTorrents() {
		if c.Request.Context().Err() != nil {
			return
		}
		rows = append(rows, torrent.LibrarySummary())
	}
	out := projectLibrary(rows, query, category, order, page, limit)
	out.SampledAt = time.Now().UTC().Format(time.RFC3339Nano)
	c.JSON(http.StatusOK, out)
}

type activeTorrent struct {
	Torrent state.TorrentStatus `json:"torrent"`
	Status  compactFlowStatus   `json:"status"`
}
type compactFlowStatus struct {
	Hash     string                   `json:"hash"`
	Startup  torr.FlowStartupStatus   `json:"startup"`
	Sessions []torr.FlowSessionStatus `json:"sessions"`
}

func flowActive(c *gin.Context) {
	items := []activeTorrent{}
	for _, torrent := range torr.ActiveTorrents() {
		if c.Request.Context().Err() != nil {
			return
		}
		row := torrent.LibrarySummary()
		if row.ActiveReaders == 0 && !row.WarmIdle && row.Stat != state.TorrentGettingInfo && row.Stat != state.TorrentPreload {
			continue
		}
		sessions := torrent.FlowStatusWithTraces(false)
		seen := map[int]bool{}
		for _, session := range sessions {
			if seen[session.FileIndex] || (session.ActiveReaders == 0 && session.State != "WARM_IDLE") {
				continue
			}
			seen[session.FileIndex] = true
			if file := torrent.PlaybackFileSummary(session.FileIndex); file != nil {
				row.FileStats = append(row.FileStats, file)
			}
		}
		items = append(items, activeTorrent{row, compactFlowStatus{row.Hash, torrent.FlowStartup(), sessions}})
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "sampled_at": time.Now().UTC().Format(time.RFC3339Nano)})
}
