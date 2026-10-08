package api

import (
	config "server/settings"
	"server/torr"
	"server/web/auth"

	"github.com/gin-gonic/gin"
)

type requestI struct {
	Action string `json:"action,omitempty"`
}

func SetupRoute(route gin.IRouter) {
	torr.StartConfigurationWorker(refreshIntegrations)
	authorized := route.Group("/", auth.CheckAuth())

	authorized.GET("/shutdown", shutdown)
	authorized.GET("/shutdown/*reason", shutdown)

	authorized.POST("/settings", settings)
	authorized.POST("/flow/playback-link", playbackLink)
	route.GET("/flow/play/:token", capabilityPlayback)
	route.HEAD("/flow/play/:token", capabilityPlayback)
	authorized.GET("/waf", getWAF)
	authorized.POST("/waf", updateWAF)
	authorized.POST("/torznab/test", torznabTest)
	authorized.POST("/jacred/test", jacredTest)

	authorized.POST("/torrents", torrents)

	authorized.POST("/torrent/upload", torrentUpload)

	authorized.POST("/cache", cache)

	route.HEAD("/stream", stream)
	route.GET("/stream", stream)

	route.HEAD("/stream/*fname", stream)
	route.GET("/stream/*fname", stream)

	route.HEAD("/play/:hash/:id", play)
	route.GET("/play/:hash/:id", play)

	authorized.POST("/viewed", viewed)

	authorized.GET("/playlistall/all.m3u", allPlayList)

	route.GET("/playlist", playList)
	route.GET("/playlist/*fname", playList)

	authorized.GET("/download/:size", download)

	if config.SearchWA {
		route.GET("/search/*query", rutorSearch)
	} else {
		authorized.GET("/search/*query", rutorSearch)
	}

	if config.SearchWA {
		route.GET("/torznab/search/*query", torznabSearch)
		route.GET("/torznab/caps", torznabCaps)
	} else {
		authorized.GET("/torznab/search/*query", torznabSearch)
		authorized.GET("/torznab/caps", torznabCaps)
	}

	if config.SearchWA {
		route.GET("/jacred/search/*query", jacredSearch)
	} else {
		authorized.GET("/jacred/search/*query", jacredSearch)
	}

	// Add storage settings endpoints
	authorized.GET("/storage/settings", GetStorageSettings)
	authorized.POST("/storage/settings", UpdateStorageSettings)

	// Add TMDB settings endpoint
	authorized.GET("/tmdb/settings", tmdbSettings)

	authorized.GET("/gst/settings", GetGStreamerSettings)
	authorized.POST("/gst/settings", UpdateGStreamerSettings)

	// Structured server status (integration flags + BT stats + raw /stat text).
	authorized.GET("/runtime/status", runtimeStatus)
	authorized.GET("/flow/status/:hash", flowStatus)
	authorized.GET("/flow/network", flowNetwork)
	authorized.GET("/flow/lan-test", flowLANTest)
	authorized.GET("/flow/tray", flowTray)
	authorized.POST("/flow/maintenance", flowMaintenance)
	authorized.GET("/flow/support", supportReport)
	authorized.GET("/flow/backup", portableBackup)
	authorized.POST("/flow/backup/preview", backupPreview)
	authorized.POST("/flow/backup/apply", backupApply)
	authorized.POST("/flow/control", flowControl)
	authorized.GET("/flow/preparation", preparationStatus)
	authorized.POST("/flow/preparation", preparationControl)
	authorized.GET("/flow/sources/:hash", webSeedStatus)
	authorized.POST("/flow/sources/:hash", webSeedControl)

	authorized.GET("/ffp/status", ffprobeStatus)
	authorized.GET("/ffp/:hash/:id", ffp)
}
