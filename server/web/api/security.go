package api

import (
	"github.com/gin-gonic/gin"
	"net"
	"net/http"
	"net/url"
	"os"
	"server/flow"
	"server/netpolicy"
	sets "server/settings"
	"server/torr"
	linkutils "server/web/api/utils"
	"strconv"
	"strings"
	"time"
)

var managementLimiter flow.ManagementLimiter
var playbackSigner, playbackSignerError = flow.NewCapabilitySigner()

func managementPath(path string) bool {
	return path != "/echo" && !strings.HasPrefix(path, "/stream") && !strings.HasPrefix(path, "/play/") && !strings.HasPrefix(path, "/flow/play/") && path != "/playlist" && !strings.HasPrefix(path, "/playlist/") && !strings.HasPrefix(path, "/assets/") && path != "/" && path != "/favicon.ico"
}

func playbackImportsSource(c *gin.Context) bool {
	path := c.Request.URL.Path
	if strings.HasPrefix(path, "/stream") || path == "/playlist" || strings.HasPrefix(path, "/playlist/") {
		if _, save := c.GetQuery("save"); save {
			return true
		}
		return linkutils.TestLink(linkutils.DecodeQueryLink(c.Query("link")), false) != nil
	}
	if strings.HasPrefix(path, "/play/") {
		return linkutils.TestLink(linkutils.DecodeQueryLink(c.Param("hash")), false) != nil
	}
	return false
}

// Install before CORS: even an OPTIONS request must pass an enabled origin
// policy. Originless native clients retain their existing authentication path.
func ManagementPolicy() gin.HandlerFunc {
	return func(c *gin.Context) {
		f := sets.CurrentFlow()
		if !managementPath(c.Request.URL.Path) && !playbackImportsSource(c) {
			c.Next()
			return
		}
		origin := c.GetHeader("Origin")
		if (origin != "" || c.GetHeader("Sec-Fetch-Site") != "") && !browserManagementHost(c.Request.Host, f.ManagementOrigins) {
			c.AbortWithStatusJSON(403, gin.H{"error": "browser management hostname requires an explicit allowed origin"})
			return
		}
		// Browser management access always needs an explicit origin grant.
		// Native players without browser headers retain their authentication path.
		if origin != "" {
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
		} else if site := c.GetHeader("Sec-Fetch-Site"); site == "cross-site" || site == "same-site" {
			// Navigations such as an image/link to GET /shutdown can omit Origin.
			c.AbortWithStatusJSON(403, gin.H{"error": "cross-origin management request requires an allowed Origin"})
			return
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

// Prevent a browser's attacker-controlled DNS name from rebinding to the LAN
// server while keeping IP URLs, the machine's local name and approved proxies.
func browserManagementHost(host, origins string) bool {
	u, err := url.Parse("http://" + host)
	if err != nil || u.User != nil || u.Host != host {
		return false
	}
	name := strings.TrimSuffix(strings.ToLower(u.Hostname()), ".")
	if net.ParseIP(name) != nil || name == "localhost" {
		return true
	}
	local, _ := os.Hostname()
	local = strings.ToLower(strings.TrimSuffix(local, ".local"))
	if local != "" && (name == local || name == local+".local") {
		return true
	}
	for _, raw := range strings.Split(origins, ",") {
		approved, err := url.Parse(strings.TrimSpace(raw))
		if err == nil && strings.EqualFold(approved.Host, host) {
			return true
		}
	}
	return false
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
	return gin.H{"http_auth": sets.HttpAuth, "legacy_playback": !f.RequirePlaybackToken, "security_profile": f.SecurityProfile, "management_rate_limit": f.ManagementRateLimit, "explicit_origin_policy": true, "listen_addresses": sets.IPs, "http_port": sets.Port, "https": sets.Ssl, "torrent_interface": policy, "windows_guard": netpolicy.GuardSnapshot(), "vpn_leak_protection_verified": false}
}
