package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"server/torr"
)

func preparationStatus(c *gin.Context) { c.JSON(http.StatusOK, torr.PreparationSnapshot()) }
func preparationControl(c *gin.Context) {
	var req struct {
		Hash      string `json:"hash" binding:"required"`
		FileIndex int    `json:"file_index"`
		Action    string `json:"action" binding:"required"`
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4096)
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid preparation request"})
		return
	}
	if err := torr.PrepareEpisode(req.Hash, req.FileIndex, req.Action); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, torr.PreparationSnapshot())
}
