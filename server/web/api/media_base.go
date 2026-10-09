package api

import (
	"net"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"server/settings"
	"server/utils"
	"server/web/sslcerts"
)

// mediaBaseURL is for external players. Browser playback retains its HTTPS
// origin to avoid mixed content. Only an available plain-media listener and a
// managed self-signed identity permit the compatibility fallback.
func mediaBaseURL(c *gin.Context) string {
	if s := settings.BTsets(); c.Request.TLS != nil && s != nil && settings.PlainHTTPServesMedia() && settings.Port != "" && sslcerts.IsGenerated(s.SslCert, s.SslKey) {
		host, _, err := net.SplitHostPort(c.Request.Host)
		if err != nil {
			host = strings.Trim(c.Request.Host, "[]")
		}
		return "http://" + net.JoinHostPort(host, settings.Port)
	}
	return utils.GetScheme(c) + "://" + utils.GetHost(c)
}

// mediaBase godoc
// @Summary Get the external-player media origin
// @Tags API
// @Security BasicAuth
// @Produce json
// @Success 200 {object} mediaBaseResponse
// @Router /mediabase [get]
func mediaBase(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, mediaBaseResponse{Base: mediaBaseURL(c)})
}

type mediaBaseResponse struct {
	Base string `json:"base"`
}
