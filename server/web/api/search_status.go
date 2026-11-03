package api

import (
	"github.com/gin-gonic/gin"
	"server/searchstatus"
	sets "server/settings"
	"server/torznab"
	"strings"
	"time"
)

func searchStatus(c *gin.Context) {
	rows := []gin.H{}
	cfg := sets.BTsets()
	add := func(source string, index int, name, host, key string) {
		if len(rows) >= 128 {
			return
		}
		if len(name) > 128 {
			name = name[:128]
		}
		row := gin.H{"source": source, "index": index, "name": strings.ToValidUTF8(name, ""), "history": searchstatus.History(searchstatus.EndpointKey(source, host, key))}
		if source == "torznab" {
			row["capabilities"] = torznab.PeekCapabilities(host, key)
		}
		rows = append(rows, row)
	}
	if cfg != nil {
		if cfg.EnableRutorSearch {
			add("rutor", -1, "Rutor", "", "")
		}
		if cfg.EnableJacRedSearch {
			add("jacred", -1, "JacRed", cfg.JacRedUrl, cfg.JacRedKey)
		}
		if cfg.EnableTorznabSearch {
			for index, endpoint := range cfg.TorznabUrls {
				name := endpoint.Name
				if name == "" {
					name = "Torznab"
				}
				add("torznab", index, name, endpoint.Host, endpoint.Key)
			}
		}
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(200, gin.H{"schema_version": 1, "sampled_at": time.Now().UTC(), "items": rows})
}
