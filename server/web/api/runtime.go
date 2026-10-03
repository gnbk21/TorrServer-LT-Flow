package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"server/diagnostics"
	sets "server/settings"
	"server/torr"
	"server/torr/storage/torrstor"
)

// RuntimeStatus is a read-only snapshot for the web UI: integration flags plus
// structured BitTorrent client stats (and the raw /stat text).
type RuntimeStatus struct {
	DLNAEnabled    bool   `json:"dlna_enabled"`
	BonjourEnabled bool   `json:"bonjour_enabled"`
	FriendlyName   string `json:"friendly_name"`
	WebDAVEnabled  bool   `json:"webdav_enabled"`
	WebDAVPath     string `json:"webdav_path"`
	FusePath       string `json:"fuse_path"`
	FuseEnabled    bool   `json:"fuse_enabled"`

	// BitTorrent client (structured alternative to plaintext GET /stat).
	BT              *torr.ClientStatusSnapshot `json:"bt,omitempty"`
	Memory          diagnostics.MemoryStatus   `json:"memory"`
	CacheAllocation torrstor.AllocationStatus  `json:"cache_allocation"`
	Startup         diagnostics.StartupStatus  `json:"startup"`
}

// runtimeStatus godoc
//
//	@Summary		Runtime integration + BitTorrent status
//	@Description	Read-only flags for DLNA/Bonjour/WebDAV/FUSE plus structured BT stats (and raw /stat text).
//	@Tags			API
//	@Produce		json
//	@Security		BasicAuth
//	@Success		200	{object}	RuntimeStatus
//	@Router			/runtime/status [get]
func runtimeStatus(c *gin.Context) {
	friendly := ""
	dlna := false
	bonjour := false
	if sets.BTsets() != nil {
		friendly = sets.BTsets().FriendlyName
		dlna = sets.BTsets().EnableDLNA
		bonjour = sets.BTsets().EnableBonjour
	}

	fusePath := ""
	webdav := false
	if sets.Args != nil {
		fusePath = sets.Args.FusePath
		webdav = sets.Args.WebDAV
	}

	bt := torr.SnapshotClientStatus()
	c.JSON(http.StatusOK, RuntimeStatus{
		DLNAEnabled:     dlna,
		BonjourEnabled:  bonjour,
		FriendlyName:    friendly,
		WebDAVEnabled:   webdav,
		WebDAVPath:      "/dav",
		FusePath:        fusePath,
		FuseEnabled:     fusePath != "",
		BT:              &bt,
		Memory:          diagnostics.Memory(),
		CacheAllocation: torrstor.Global().Allocations(),
		Startup:         diagnostics.Startup(),
	})
}
