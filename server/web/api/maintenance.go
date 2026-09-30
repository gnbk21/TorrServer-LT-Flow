package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"server/flow"
	"server/torr"
	"time"
)

func flowMaintenance(c *gin.Context) {
	if !sameOriginMaintenance(c) {
		return
	}
	var req struct {
		Enabled bool   `json:"enabled"`
		Token   string `json:"token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid maintenance request"})
		return
	}
	if req.Enabled && torr.FlowHasActiveWork() {
		c.JSON(http.StatusConflict, gin.H{"error": "playback or preload is active"})
		return
	}
	if !req.Enabled {
		if !flow.Maintenance.Release(req.Token) {
			c.JSON(http.StatusConflict, gin.H{"error": "maintenance lease does not match"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"enabled": false})
		return
	}
	token, active := flow.Maintenance.Acquire(120 * time.Second)
	if token == "" {
		c.JSON(http.StatusConflict, gin.H{"error": "requests are active", "active_requests": active})
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"enabled": true, "active_requests": active, "token": token, "lease_seconds": 120})
}
