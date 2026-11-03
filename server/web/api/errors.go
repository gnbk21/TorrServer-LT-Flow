package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"server/log"
	"strings"
)

// abortWithJSONError aborts the request with {"error": "..."} as the body.
// gin's AbortWithError only sets the status and records the error, leaving the
// response empty, which breaks clients that parse every reply as JSON.
func abortWithJSONError(c *gin.Context, code int, err error) {
	abortWithErrorCode(c, code, httpErrorCode(code), err)
}

// Machine-readable codes are additive; legacy clients still receive error text
// and the original HTTP status. Starting requires explicit engine-state evidence.
func abortWithErrorCode(c *gin.Context, status int, code string, err error, schemaVersion ...int) {
	body := gin.H{"error": errorMessage(err), "code": code}
	// New versioned interfaces opt in; legacy error maps retain string-only
	// values for old clients. Status, code and redacted text stay unchanged.
	if len(schemaVersion) > 0 {
		body["schema_version"] = schemaVersion[0]
		c.Header("Cache-Control", "private, no-store")
		if c.Request != nil && c.Request.Method == http.MethodHead {
			c.Header("Content-Type", "application/json; charset=utf-8")
			c.AbortWithStatus(status)
			return
		}
	}
	c.AbortWithStatusJSON(status, body)
}

func httpErrorCode(status int) string {
	switch status {
	case http.StatusUnauthorized:
		return "AUTH_REQUIRED"
	case http.StatusForbidden:
		return "ACCESS_FORBIDDEN"
	case http.StatusTooManyRequests:
		return "RATE_LIMITED"
	case http.StatusServiceUnavailable:
		return "SERVICE_UNAVAILABLE"
	case http.StatusConflict:
		return "STATE_CONFLICT"
	case http.StatusBadRequest:
		return "INVALID_REQUEST"
	case http.StatusNotFound:
		return "NOT_FOUND"
	default:
		return "REQUEST_FAILED"
	}
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
