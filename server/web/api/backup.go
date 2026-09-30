package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"server/diagnostics"
	"server/flow"
	sets "server/settings"
	"server/torr"
)

var backupMu sync.Mutex

func portableBackup(c *gin.Context) {
	backup, err := sets.ExportBackup()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	data, err := json.MarshalIndent(backup, "", "  ")
	if err != nil || len(data) > sets.BackupLimit {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "backup exceeds export limit"})
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Header("Content-Disposition", `attachment; filename="TorrServer-Flow-backup.json"`)
	c.Data(http.StatusOK, "application/json", data)
}

func sameOriginMaintenance(c *gin.Context) bool {
	origin := c.GetHeader("Origin")
	if origin == "" {
		return true
	}
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Host != c.Request.Host || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		c.AbortWithStatus(http.StatusForbidden)
		return false
	}
	return true
}

func backupPreview(c *gin.Context) {
	if !sameOriginMaintenance(c) {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, sets.BackupLimit)
	data, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "backup exceeds import limit"})
		return
	}
	backup, err := sets.ParseBackup(data)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	keys := []string{}
	for name := range backup.Settings {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"digest": backup.Digest(), "library_count": len(backup.Library), "settings_fields": keys, "credentials_excluded": true, "mode": "merge", "restart_required": true})
}

func backupApply(c *gin.Context) {
	if !sameOriginMaintenance(c) {
		return
	}
	backupMu.Lock()
	defer backupMu.Unlock()
	if sets.ReadOnly {
		c.JSON(http.StatusConflict, gin.H{"error": "database is read-only"})
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, sets.BackupLimit+4096)
	var request struct {
		Backup json.RawMessage `json:"backup"`
		Digest string          `json:"digest"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid restore request"})
		return
	}
	backup, err := sets.ParseBackup(request.Backup)
	if err != nil || backup.Digest() != request.Digest {
		c.JSON(http.StatusBadRequest, gin.H{"error": "backup does not match the preview"})
		return
	}
	if torr.FlowHasActiveWork() {
		c.JSON(http.StatusConflict, gin.H{"error": "stop playback and preload before restoring"})
		return
	}
	token, active := flow.Maintenance.Acquire(0)
	if token == "" || active != 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "server is busy"})
		return
	}
	defer flow.Maintenance.Release(token)
	recovery := sets.CaptureRecovery()
	recoveryData, err := json.Marshal(recovery)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot capture recovery backup"})
		return
	}
	directory := filepath.Join(sets.Path, "flow-backups")
	if err = os.MkdirAll(directory, 0700); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot create recovery directory"})
		return
	}
	name := "before-restore-" + time.Now().UTC().Format("20060102T150405.000000000") + ".json"
	if err = diagnostics.WritePrivateFile(filepath.Join(directory, name), recoveryData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot save protected recovery backup"})
		return
	}
	merged, err := backup.MergedSettings()
	if err == nil {
		err = torr.SetSettings(merged)
	}
	if err == nil {
		err = sets.WriteLibraryBatch(backup.MergedLibrary(recovery.Library), false)
	}
	if err != nil {
		settingsErr := torr.SetSettings(recovery.Settings)
		libraryErr := sets.WriteLibraryBatch(recovery.Library, true)
		if settingsErr != nil || libraryErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "restore and recovery failed; use the local pre-apply backup", "recovery_backup": name})
			return
		}
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "restore failed; previous settings and library were restored", "recovery_backup": name})
		return
	}
	c.JSON(http.StatusOK, gin.H{"restored": true, "library_count": len(backup.Library), "recovery_backup": name})
}
