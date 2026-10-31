package api

import (
	"github.com/gin-gonic/gin"
	"server/torr"
	"strconv"
)

func cachedMediaInfo(c *gin.Context) {
	index, err := strconv.Atoi(c.Param("index"))
	if err != nil || index < 1 {
		c.Status(400)
		return
	}
	t := torr.GetTorrentInfo(c.Param("hash"))
	if t == nil {
		c.Status(404)
		return
	}
	info, known := t.CachedMediaInfo(index)
	c.Header("Cache-Control", "no-store")
	c.JSON(200, gin.H{"schema_version": 1, "known": known, "source": "existing_bounded_probe", "media": info})
}
