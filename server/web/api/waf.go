package api

import (
	"net/http"

	sets "server/settings"
	"server/web/waf"

	"github.com/gin-gonic/gin"
)

type wafResponse struct {
	Whitelist              string        `json:"whitelist"`
	Blacklist              string        `json:"blacklist"`
	Referers               string        `json:"referers"`
	DefaultReferers        []string      `json:"default_referers"`
	DefaultReferersEnabled bool          `json:"default_referers_enabled"`
	IPEnabled              bool          `json:"ip_enabled"`
	RefererEnabled         bool          `json:"referer_enabled"`
	ReadOnly               bool          `json:"read_only"`
	Warnings               []waf.Warning `json:"warnings"`
}

type wafUpdateReq struct {
	Whitelist *string `json:"whitelist" binding:"required"`
	Blacklist *string `json:"blacklist" binding:"required"`
	Referers  *string `json:"referers" binding:"required"`
	// Optional: omitted keeps the current state of the built-in denylist.
	DefaultReferersEnabled *bool `json:"default_referers_enabled"`
}

func snapshotToResponse(snap waf.Snapshot) wafResponse {
	warnings := snap.Warnings
	if warnings == nil {
		warnings = []waf.Warning{}
	}
	return wafResponse{
		Whitelist:              snap.WhitelistText,
		Blacklist:              snap.BlacklistText,
		Referers:               snap.ReferersText,
		DefaultReferers:        waf.DefaultReferers(),
		DefaultReferersEnabled: snap.DefaultReferersEnabled,
		IPEnabled:              snap.IPEnabled,
		RefererEnabled:         snap.RefererEnabled,
		ReadOnly:               sets.ReadOnly,
		Warnings:               warnings,
	}
}

// getWAF godoc
//
//	@Summary		Get HTTP WAF lists
//	@Description	Returns whitelist/blacklist IP and blocked referer lists stored in settings.json, plus the built-in referer denylist and whether it is enforced.
//
//	@Tags			API
//	@Produce		json
//	@Security		BasicAuth
//	@Success		200	{object}	wafResponse
//	@Failure		401	{object}	map[string]string
//	@Router			/waf [get]
func getWAF(c *gin.Context) {
	c.JSON(http.StatusOK, snapshotToResponse(waf.GetSnapshot()))
}

// updateWAF godoc
//
//	@Summary		Update HTTP WAF lists
//	@Description	Fully replaces whitelist/blacklist IP and blocked referer lists in settings.json and hot-reloads the WAF. All three lists are required; explicit empty strings clear a list. default_referers_enabled is optional and switches the built-in referer denylist.
//
//	@Tags			API
//	@Accept			json
//	@Produce		json
//	@Security		BasicAuth
//	@Param			request	body		wafUpdateReq	true	"Complete WAF configuration"
//	@Success		200		{object}	wafResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		403		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Router			/waf [post]
func updateWAF(c *gin.Context) {
	if sets.ReadOnly {
		c.JSON(http.StatusForbidden, gin.H{"error": "Read-only mode"})
		return
	}

	var req wafUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": errorMessage(err)})
		return
	}

	snap, err := waf.Update(waf.ListsUpdate{
		Whitelist:              *req.Whitelist,
		Blacklist:              *req.Blacklist,
		Referers:               *req.Referers,
		DefaultReferersEnabled: req.DefaultReferersEnabled,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": errorMessage(err)})
		return
	}
	c.JSON(http.StatusOK, snapshotToResponse(snap))
}
