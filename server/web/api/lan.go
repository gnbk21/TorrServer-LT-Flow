package api

import (
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

var lanTestSlots = make(chan struct{}, 2)

// flowLANTest measures only server-to-browser transport. It has no torrent,
// arbitrary target URL, persistent result or background work after the request.
func flowLANTest(c *gin.Context) {
	mb := 16
	if value := c.Query("mib"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 32 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "mib must be 1..32"})
			return
		}
		mb = parsed
	}
	select {
	case lanTestSlots <- struct{}{}:
		defer func() { <-lanTestSlots }()
	default:
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "transfer test already busy"})
		return
	}
	control := http.NewResponseController(c.Writer)
	if err := control.SetWriteDeadline(time.Now().Add(15 * time.Second)); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "bounded transfer deadline unavailable"})
		return
	}
	defer control.SetWriteDeadline(time.Time{})
	size := int64(mb) << 20
	c.Header("Cache-Control", "no-store, no-transform")
	c.Header("Content-Type", "application/octet-stream")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Content-Length", strconv.FormatInt(size, 10))
	// The fixed scratch buffer bounds memory independent of transfer size.
	_, _ = io.CopyBuffer(c.Writer, newFR(size), make([]byte, 64<<10))
}
