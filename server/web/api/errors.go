package api

import (
	"github.com/gin-gonic/gin"
	"server/log"
	"strings"
)

// abortWithJSONError aborts the request with {"error": "..."} as the body.
// gin's AbortWithError only sets the status and records the error, leaving the
// response empty, which breaks clients that parse every reply as JSON.
func abortWithJSONError(c *gin.Context, code int, err error) {
	c.AbortWithStatusJSON(code, gin.H{"error": errorMessage(err)})
}

func errorMessage(err error) string {
	if err == nil {
		return "request failed"
	}
	message := log.RedactSecrets(err.Error())
	if len(message) > 4096 {
		message = strings.ToValidUTF8(message[:4096], "") + "…"
	}
	return message
}
