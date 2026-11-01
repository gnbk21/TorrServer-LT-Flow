package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"server/diagnostics"
	"server/netbind"
	sets "server/settings"
	"time"
)

func pairingCreate(c *gin.Context) {
	p, err := diagnostics.Pairing.Create(diagnostics.PairingRequestPeer(c.Request), time.Now())
	if err != nil {
		c.Header("Retry-After", "300")
		abortWithJSONError(c, 429, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	addresses, _ := netbind.LocalIPs()
	c.JSON(http.StatusCreated, gin.H{"schema_version": 1, "proof": p, "startup": diagnostics.Startup(), "listener_addresses": netbind.Normalize(sets.IPs), "advertised_addresses": addresses, "auth_required": sets.HttpAuth})
}
func pairingStatus(c *gin.Context) {
	p, ok := diagnostics.Pairing.Status(c.Param("id"), time.Now())
	if !ok {
		c.Status(http.StatusGone)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, p)
}
func pairingVisit(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("Referrer-Policy", "no-referrer")
	c.Header("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
	c.Header("X-Content-Type-Options", "nosniff")
	if !diagnostics.Pairing.Visit(c.Param("token"), diagnostics.PairingRequestPeer(c.Request), time.Now()) {
		c.String(410, "Pairing proof expired or unavailable. Create a new proof on the PC.")
		return
	}
	c.String(200, "FLOW_CONNECTION_RECEIVED\nReturn to the Flow setup on your PC. This read-only proof grants no management or playback access.")
}
