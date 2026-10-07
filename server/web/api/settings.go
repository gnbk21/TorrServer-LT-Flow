package api

import (
	"net/http"

	"server/rutor"

	"server/bonjour"
	"server/dlna"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"

	sets "server/settings"
	"server/torr"
)

// Action: get, set, def
type setsReqJS struct {
	requestI
	Sets     *sets.BTSets `json:"sets,omitempty"`
	Revision string       `json:"revision,omitempty"`
	When     string       `json:"when,omitempty"`
}

// settings godoc
//
//	@Summary		Get / Set server settings
//	@Description	Allow to get or set server settings.
//
//	@Tags			API
//
//	@Param			request	body	setsReqJS	true	"Settings request. Available params for action: get, set, def"
//
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	sets.BTSets	"Settings JSON or nothing. Depends on what action has been asked."
//	@Router			/settings [post]
func settings(c *gin.Context) {
	var req setsReqJS
	err := c.ShouldBindJSON(&req)
	if err != nil {
		abortWithJSONError(c, http.StatusBadRequest, err)
		return
	}

	if req.Action == "get" {
		c.JSON(200, sets.BTsets())
		return
	} else if req.Action == "state" {
		c.Header("Cache-Control", "no-store")
		c.JSON(200, torr.ConfigurationSnapshot())
		return
	} else if req.Action == "plan" {
		if err := sets.ValidateSettings(req.Sets); err != nil {
			abortWithJSONError(c, 400, err)
			return
		}
		c.JSON(200, gin.H{"restart_required": sets.NeedsEngineRestart(sets.BTsets(), sets.NormalizeConfiguration(req.Sets)), "active_work": torr.FlowHasActiveWork()})
		return
	} else if req.Action == "cancel_pending" {
		if err := torr.CancelPendingConfiguration(req.Revision); err != nil {
			abortWithJSONError(c, 409, err)
			return
		}
		c.Status(200)
		return
	} else if req.Action == "set" {
		if req.Sets == nil {
			abortWithJSONError(c, http.StatusBadRequest, errors.New("sets is required"))
			return
		}
		if req.When == "" {
			req.When = "now"
		}
		if err := torr.ApplyConfiguration(req.Sets, req.Revision, req.When); err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, torr.ErrSettingsConflict) {
				status = http.StatusConflict
			}
			abortWithJSONError(c, status, err)
			return
		}
		c.Status(200)
		return
	} else if req.Action == "def" {
		if err := torr.ApplyConfiguration(sets.NewDefaultConfig(), req.Revision, "now"); err != nil {
			abortWithJSONError(c, 409, err)
			return
		}
		c.Status(200)
		return
	}
	abortWithJSONError(c, http.StatusBadRequest, errors.New("action is empty"))
}

func refreshIntegrations(s *sets.BTSets) {
	dlna.Stop()
	if s.EnableDLNA {
		dlna.Start()
	}
	bonjour.Stop()
	if s.EnableBonjour {
		bonjour.Start()
	}
	rutor.Stop()
	rutor.Start()
}
