package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"server/flow"
	"server/torr"
)

func flowMaintenance(c *gin.Context) {
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid maintenance request"})
		return
	}
	if req.Enabled && torr.FlowHasActiveWork() {
		c.JSON(http.StatusConflict, gin.H{"error": "playback or preload is active"})
		return
	}
	enabled, active := flow.Maintenance.Set(req.Enabled)
	if req.Enabled && !enabled {
		c.JSON(http.StatusConflict, gin.H{"error": "requests are active", "active_requests": active})
		return
	}
	c.JSON(http.StatusOK, gin.H{"enabled": enabled, "active_requests": active})
}
