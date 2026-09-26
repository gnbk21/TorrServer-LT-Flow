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
	c.JSON(http.StatusOK, gin.H{"hash": c.Param("hash"), "startup": t.FlowStartup(), "sessions": t.FlowStatus(), "trackers": t.FlowTrackers()})
}

func flowNetwork(c *gin.Context) {
	c.JSON(http.StatusOK, torr.NetworkStatusSnapshot())
}

func flowTray(c *gin.Context) {
	c.JSON(http.StatusOK, torr.SnapshotFlowTray())
}

func flowControl(c *gin.Context) {
	var request struct {
		Action string `json:"action"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var paused bool
	switch request.Action {
	case "pause":
		paused = true
	case "resume":
		paused = false
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown Flow action"})
		return
	}
	if err := torr.SetFlowPaused(paused); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, torr.SnapshotFlowTray())
}
