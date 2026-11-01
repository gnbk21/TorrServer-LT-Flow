package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"server/diagnostics"
)

// Raw traces may contain process details and require explicit selection. They
// are intentionally absent from supportReport and every ordinary backup/export.
func incidentRaw(c *gin.Context) {
	f, err := diagnostics.Incidents.OpenRaw(c.Param("name"))
	if err != nil {
		abortWithJSONError(c, http.StatusNotFound, err)
		return
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || st.Size() > diagnostics.IncidentOutputLimit {
		c.Status(404)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", `attachment; filename="Flow-private-incident.trace"`)
	http.ServeContent(c.Writer, c.Request, "incident.trace", st.ModTime(), f)
}
