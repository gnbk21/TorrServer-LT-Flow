package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"server/torr"
)

func webSeedStatus(c *gin.Context) {
	status, err := torr.TorrentWebSeeds(c.Param("hash"))
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, status)
}
func webSeedControl(c *gin.Context) {
	var req struct {
		Action     string `json:"action"`
		URL        string `json:"url"`
		ID         string `json:"id"`
		AllowLocal bool   `json:"allow_local"`
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 16384)
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid mirror request"})
		return
	}
	if err := torr.UpdateTorrentWebSeed(c.Param("hash"), req.Action, req.URL, req.ID, req.AllowLocal); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	status, err := torr.TorrentWebSeeds(c.Param("hash"))
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, status)
}
