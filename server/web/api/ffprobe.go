package api

import (
	"fmt"
	"net/http"

	"server/ffprobe"
	"server/log"
	sets "server/settings"
	"server/torr"

	"github.com/gin-gonic/gin"
)

type ffprobeStatusResponse struct {
	Available bool `json:"available"`
}

// ffprobeStatus godoc
//
//	@Summary		ffprobe availability
//	@Description	Reports whether the ffprobe binary is available. Features that need the real
//	@Description	media duration (e.g. saving the playback position) work only when it is.
//
//	@Tags			API
//
//	@Produce		json
//	@Success		200	{object}	ffprobeStatusResponse
//	@Router			/ffp/status [get]
func ffprobeStatus(c *gin.Context) {
	c.JSON(http.StatusOK, ffprobeStatusResponse{Available: ffprobe.Exists()})
}

// ffp godoc
//
//	@Summary		Gather informations using ffprobe
//	@Description	Gather informations using ffprobe.
//
//	@Tags			API
//
//	@Param			hash	path	string	true	"Torrent hash"
//	@Param			id		path	string	true	"File index in torrent"
//
//	@Produce		json
//	@Success		200	"Data returned from ffprobe"
//	@Router			/ffp/{hash}/{id} [get]
func ffp(c *gin.Context) {
	hash := c.Param("hash")
	indexStr := c.Param("id")

	if hash == "" || indexStr == "" {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "link should not be empty"})
		return
	}

	if !ffprobe.Exists() {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "ffprobe binary not found"})
		return
	}

	link := sets.LoopbackBaseURL() + "/play/" + hash + "/" + indexStr

	data, err := ffprobe.ProbeURLContext(c.Request.Context(), torr.InternalMediaURL(link, true))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": log.RedactInternalMediaKey(fmt.Sprintf("error getting data: %v", err))})
		return
	}

	if data.Format != nil {
		data.Format.Filename = "/play/" + hash + "/" + indexStr
	}
	c.JSON(200, data)
}
