package api

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"net/http"
	"runtime"
	"server/diagnostics"
	"server/ffprobe"
	"server/lt"
	"server/settings"
	"server/torr"
	"server/torr/storage/torrstor"
	"server/version"
	"time"
)

func supportReport(c *gin.Context) {
	playback, truncated := torr.SupportPlayback()
	network := torr.NetworkStatusSnapshot()
	network.Addresses = nil
	network.LastError = ""
	report := gin.H{
		"schema_version": 1, "created_at": time.Now().UTC(), "version": version.Version, "go_version": runtime.Version(), "libtorrent_version": lt.Version(), "os": runtime.GOOS, "architecture": runtime.GOARCH,
		"memory": diagnostics.Memory(), "cache_allocation": torrstor.Global().Allocations(), "startup": diagnostics.Startup(), "network": network,
		"flow_settings": settings.CurrentFlow(), "ffprobe_available": ffprobe.Exists(), "http_auth_enabled": settings.HttpAuth,
		"playback": playback, "truncated": truncated, "redaction": "Names, hashes, paths, URLs, accounts, client tokens, logs and traces are excluded. Numeric file sizes and indices remain. Cache counters are per torrent and shared between sessions.",
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil || len(data) > 2<<20 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "support report exceeds safe export bounds"})
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Header("Content-Disposition", `attachment; filename="TorrServer-Flow-support.json"`)
	c.Data(http.StatusOK, "application/json", data)
}
