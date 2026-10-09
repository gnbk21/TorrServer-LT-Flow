package api

import (
	"github.com/gin-gonic/gin"
	"net"
	"net/http"
	"net/url"
	"server/flow"
	"server/netpolicy"
	sets "server/settings"
	"server/torr"
	"strconv"
	"strings"
	"time"
)

var managementLimiter flow.ManagementLimiter
var playbackSigner, playbackSignerError = flow.NewCapabilitySigner()

func managementPath(path string) bool {
	return path != "/echo" && !strings.HasPrefix(path, "/stream") && !strings.HasPrefix(path, "/play/") && !strings.HasPrefix(path, "/flow/play/") && !strings.HasPrefix(path, "/playlist") && !strings.HasPrefix(path, "/assets/") && path != "/" && path != "/favicon.ico"
}

// Install before CORS: even an OPTIONS request must pass an enabled origin
// policy. Originless native clients retain their existing authentication path.
func ManagementPolicy() gin.HandlerFunc {
	return func(c *gin.Context) {
		f := sets.CurrentFlow()
		if !managementPath(c.Request.URL.Path) {
			c.Next()
			return
		}
		origin := c.GetHeader("Origin")
		if origin != "" && (f.SecurityProfile == "restricted" || f.ManagementOrigins != "") {
			scheme := "http"
			if c.Request.TLS != nil {
				scheme = "https"
			}
			allowed := origin == scheme+"://"+c.Request.Host
			for _, s := range strings.Split(f.ManagementOrigins, ",") {
				if origin == strings.TrimSuffix(strings.TrimSpace(s), "/") {
					allowed = true
				}
			}
			if !allowed {
				c.AbortWithStatusJSON(403, gin.H{"error": "management origin is not allowed"})
				return
			}
		}
		if c.Request.Method != http.MethodOptions {
			key, _, err := net.SplitHostPort(c.Request.RemoteAddr)
			if err != nil {
				key = c.Request.RemoteAddr
			}
			if !managementLimiter.Allow(key, f.ManagementRateLimit, time.Now()) {
				c.Header("Retry-After", "60")
				c.AbortWithStatusJSON(429, gin.H{"error": "management request limit reached"})
				return
			}
		}
		c.Next()
	}
}

func PlaybackPolicy() gin.HandlerFunc {
	return func(c *gin.Context) {
		p := c.Request.URL.Path
		legacy := strings.HasPrefix(p, "/stream") || strings.HasPrefix(p, "/play/") || strings.HasPrefix(p, "/playlist")
		if legacy && sets.CurrentFlow().RequirePlaybackToken && c.GetString(gin.AuthUserKey) == "" && !torr.IsInternalProbe(c.Request) {
			c.AbortWithStatusJSON(401, gin.H{"error": "a playback capability or authentication is required"})
			return
		}
		c.Next()
	}
}

func playbackLink(c *gin.Context) {
	if sets.CurrentFlow().RequirePlaybackToken && c.GetString(gin.AuthUserKey) == "" {
		c.AbortWithStatus(401)
		return
	}
	var req struct {
		Hash  string `json:"hash"`
		Index int    `json:"index"`
	}
	if c.ShouldBindJSON(&req) != nil {
		c.AbortWithStatus(400)
		return
	}
	tor := torr.GetTorrent(req.Hash)
	if tor == nil {
		c.AbortWithStatus(404)
		return
	}
	found := false
	for _, f := range tor.Status().FileStats {
		if f.Id == req.Index {
			found = true
			break
		}
	}
	if !found {
		c.AbortWithStatusJSON(409, gin.H{"error": "wait for file metadata before creating a link"})
		return
	}
	if playbackSignerError != nil {
		c.AbortWithStatus(503)
		return
	}
	expires := time.Now().Add(time.Duration(sets.CurrentFlow().PlaybackTokenTTL) * time.Second)
	token, err := playbackSigner.Mint(flow.PlaybackClaim{Hash: strings.ToLower(req.Hash), Index: req.Index, Expires: expires.Unix()}, time.Now())
	if err != nil {
		c.AbortWithStatus(400)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(200, gin.H{"path": "/flow/play/" + url.PathEscape(token), "expires_at": expires.UTC(), "revoked_on_restart": true})
}
func capabilityPlayback(c *gin.Context) {
	if playbackSignerError != nil {
		c.AbortWithStatus(503)
		return
	}
	claim, err := playbackSigner.Verify(c.Param("token"), time.Now())
	if err != nil {
		c.AbortWithStatus(401)
		return
	}
	// A capability grants one existing file only; it cannot import torrents,
	// list a library, mint another token or authorize management endpoints.
	tor := torr.GetTorrent(claim.Hash)
	if tor == nil {
		c.AbortWithStatus(404)
		return
	}
	found := false
	for _, f := range tor.Status().FileStats {
		if f.Id == claim.Index {
			found = true
			break
		}
	}
	if !found {
		c.AbortWithStatus(404)
		return
	}
	c.Header("Cache-Control", "private, no-store")
	c.Header("Referrer-Policy", "no-referrer")
	c.Params = append(c.Params, gin.Param{Key: "hash", Value: claim.Hash}, gin.Param{Key: "id", Value: strconv.Itoa(claim.Index)})
	c.Set(gin.AuthUserKey, "playback-capability")
	play(c)
}

func ExposureSummary() gin.H {
	f := sets.CurrentFlow()
	policy := netpolicy.Snapshot()
	return gin.H{"http_auth": sets.HttpAuth, "legacy_playback": !f.RequirePlaybackToken, "security_profile": f.SecurityProfile, "management_rate_limit": f.ManagementRateLimit, "explicit_origin_policy": f.ManagementOrigins != "" || f.SecurityProfile == "restricted", "listen_addresses": sets.IPs, "http_port": sets.Port, "https": sets.Ssl, "torrent_interface": policy, "vpn_leak_protection_verified": false}
}
