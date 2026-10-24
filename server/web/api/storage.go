package api

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"server/torr"

	sets "server/settings"

	"github.com/gin-gonic/gin"
)

// GetStorageSettings godoc
// @Summary Get storage configuration settings
// @Description Retrieves the current storage preferences for settings and viewed history
// @Tags API
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} map[string]interface{} "Storage preferences"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /storage/settings [get]
func GetStorageSettings(c *gin.Context) {
	prefs := sets.GetStoragePreferences()
	c.JSON(http.StatusOK, prefs)
}

// UpdateStorageSettings godoc
// @Summary Update storage configuration settings
// @Description Updates the storage preferences for settings and viewed history. Requires application restart for changes to take effect.
// @Tags API
// @Accept json,x-www-form-urlencoded
// @Produce json
// @Security ApiKeyAuth
// @Param request body map[string]interface{} true "Storage preferences to update"
// @Param settings formData string false "Settings storage type" Enums(json,bbolt)
// @Param viewed formData string false "Viewed history storage type" Enums(json,bbolt)
// @Success 200 {object} map[string]string "Update successful"
// @Failure 400 {object} map[string]string "Invalid input data"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Read-only mode"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /storage/settings [post]
func UpdateStorageSettings(c *gin.Context) {
	if sets.ReadOnly {
		c.JSON(http.StatusForbidden, gin.H{"error": "Read-only mode"})
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4096)
	prefs := map[string]interface{}{}
	media, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if err != nil && c.GetHeader("Content-Type") != "" {
		c.JSON(400, gin.H{"error": "Invalid Content-Type"})
		return
	}
	if media == "application/x-www-form-urlencoded" {
		if err = c.Request.ParseForm(); err != nil {
			c.JSON(400, gin.H{"error": "Invalid form"})
			return
		}
		for key, values := range c.Request.PostForm {
			if len(values) != 1 {
				c.JSON(400, gin.H{"error": "Duplicate preference"})
				return
			}
			prefs[key] = values[0]
		}
	} else if media == "application/json" || media == "" {
		decoder := json.NewDecoder(c.Request.Body)
		if err = decoder.Decode(&prefs); err != nil {
			c.JSON(400, gin.H{"error": "Invalid JSON"})
			return
		}
		var trailing interface{}
		if decoder.Decode(&trailing) != io.EOF {
			c.JSON(400, gin.H{"error": "Trailing JSON"})
			return
		}
	} else {
		c.JSON(415, gin.H{"error": "Use JSON or form data"})
		return
	}
	revision := ""
	if value, ok := prefs["revision"]; ok {
		var valid bool
		revision, valid = value.(string)
		if !valid {
			c.JSON(400, gin.H{"error": "Invalid revision"})
			return
		}
		delete(prefs, "revision")
	}
	edits := 0
	for key, value := range prefs {
		if key == "viewedCount" {
			delete(prefs, key)
			continue
		}
		text, ok := value.(string)
		if (key != "settings" && key != "viewed") || !ok || (text != "json" && text != "bbolt") {
			c.JSON(400, gin.H{"error": "Invalid storage preference"})
			return
		}
		edits++
	}
	if edits == 0 {
		c.JSON(400, gin.H{"error": "No editable preferences provided"})
		return
	}
	if err = torr.ApplyStoragePreferences(prefs, revision); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, torr.ErrSettingsConflict) || errors.Is(err, torr.ErrStorageBusy) {
			status = http.StatusConflict
		}
		code := "STORAGE_WRITE_FAILED"
		if errors.Is(err, torr.ErrSettingsConflict) {
			code = "SETTINGS_CONFLICT"
		}
		if errors.Is(err, torr.ErrStorageBusy) {
			code = "STORAGE_BUSY"
		}
		if sets.StorageRecoveryPending() {
			code = "STORAGE_RECOVERY_REQUIRED"
		}
		c.JSON(status, gin.H{"error": errorMessage(err), "code": code})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
