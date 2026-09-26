package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"server/torr"
)

// flowStatus exposes a bounded diagnostic snapshot for one live torrent.
// It shares the existing API authentication middleware.
func flowStatus(c *gin.Context) {
	t := torr.GetTorrent(c.Param("hash"))
	if t == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "torrent not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"hash": c.Param("hash"), "startup": t.FlowStartup(), "sessions": t.FlowStatus()})
}

func flowNetwork(c *gin.Context) {
	c.JSON(http.StatusOK, torr.NetworkStatusSnapshot())
}
