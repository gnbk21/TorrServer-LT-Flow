package template

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// assetHandler uses the standard conditional request implementation for GET
// and HEAD. Hash once when registering a route, not on each phone refresh.
func assetHandler(data []byte, contentType, cacheControl string) gin.HandlerFunc {
	etag := fmt.Sprintf(`"%x"`, sha256.Sum256(data))
	return func(c *gin.Context) {
		c.Header("Content-Type", contentType)
		c.Header("Cache-Control", cacheControl)
		c.Header("ETag", etag)
		http.ServeContent(c.Writer, c.Request, "", time.Time{}, bytes.NewReader(data))
	}
}
