package api

import (
	"net/http"
	"server/searchstatus"
	"time"

	"github.com/gin-gonic/gin"

	"server/jacred"
	"server/rutor/models"
	sets "server/settings"
	"server/web/api/utils"
)

// jacredSearch godoc
//
//	@Summary		Makes a JacRed search
//	@Description	Searches the configured JacRed aggregator.
//
//	@Tags			API
//
//	@Param			query	query	string	true	"JacRed query"
//
//	@Produce		json
//	@Success		200	{array}	models.TorrentDetails	"JacRed torrent search result(s)"
//	@Router			/jacred/search [get]
func jacredSearch(c *gin.Context) {
	if !sets.BTsets().EnableJacRedSearch {
		c.JSON(http.StatusBadRequest, []string{})
		return
	}
	query := utils.DecodeQueryValue(c.Query("query"))
	started := time.Now()
	list, err := jacred.SearchDetailed(c.Request.Context(), query)
	if list == nil {
		list = []*models.TorrentDetails{}
	}
	if c.Query("details") == "true" {
		c.JSON(200, detailedSearchResponse{list, []searchstatus.Status{searchstatus.Outcome("jacred", -1, len(list), started, err)}})
		return
	}
	c.JSON(200, list)
}

type jacredTestReq struct {
	Host string `json:"host"`
	Key  string `json:"key"`
}

func jacredTest(c *gin.Context) {
	var req jacredTestReq
	if err := c.ShouldBindJSON(&req); err != nil {
		abortWithJSONError(c, http.StatusBadRequest, err)
		return
	}

	if err := jacred.Test(c.Request.Context(), req.Host, req.Key); err != nil {
		outcome := searchstatus.Outcome("jacred", -1, 0, time.Now(), err)
		c.JSON(200, gin.H{"success": false, "error": errorMessage(err), "state": outcome.State})
		return
	}
	c.JSON(200, gin.H{"success": true})
}
