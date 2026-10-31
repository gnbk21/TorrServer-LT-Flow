package api

import (
	"context"
	"github.com/gin-gonic/gin"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"server/cachefs"
	"server/diagnostics"
	sets "server/settings"
	"server/torr/storage/torrstor"
	"strings"
	"time"
)

type storageCategory struct {
	Bytes     int64 `json:"bytes"`
	Files     int   `json:"files"`
	Truncated bool  `json:"truncated"`
	Protected bool  `json:"protected"`
}

var pieceInventoryName = regexp.MustCompile(`^(?:[a-f0-9]{40}|[a-f0-9]{64})[/\\][0-9]+$`)

// Walk only configured runtime roots, never arbitrary request paths. Reuse the
// cache's owner-selected directory capability, pinning the root for the walk.
// Descendant symlinks/junctions and nonregular files are not followed.
func storageSize(ctx context.Context, root string, accept func(string) bool, descend ...func(string) bool) storageCategory {
	out := storageCategory{Protected: true}
	if root == "" {
		return out
	}
	directory, err := cachefs.Open(root, false)
	if os.IsNotExist(err) {
		return out
	}
	if err != nil {
		out.Truncated = true
		return out
	}
	defer directory.Close()
	visited := 0
	var walk func(string, int)
	walk = func(relative string, depth int) {
		if ctx.Err() != nil || visited >= 10000 || depth >= 64 {
			out.Truncated = true
			return
		}
		// OpenRoot pins a directory, rejecting replacement FIFOs before we
		// open its contents. Replacement links cannot escape the selected root.
		subdir, err := directory.OpenRoot(relative)
		if err != nil {
			out.Truncated = true
			return
		}
		defer subdir.Close()
		reader, err := subdir.Open(".")
		if err != nil {
			out.Truncated = true
			return
		}
		defer reader.Close()
		for {
			if ctx.Err() != nil || visited >= 10000 {
				out.Truncated = true
				return
			}
			// WalkDir/ReadDir(-1) would inventory an entire large directory
			// before its callback sees the deadline or entry budget.
			entries, readErr := reader.ReadDir(min(128, 10001-visited))
			for _, entry := range entries {
				if ctx.Err() != nil || visited >= 10000 {
					out.Truncated = true
					return
				}
				visited++
				path := filepath.Join(relative, entry.Name())
				if entry.Type()&os.ModeSymlink != 0 {
					out.Truncated = true
					continue
				}
				if entry.IsDir() {
					if len(descend) == 0 || descend[0](path) {
						walk(path, depth+1)
					}
					continue
				}
				if !accept(path) {
					continue
				}
				info, err := entry.Info()
				if err != nil {
					out.Truncated = true
				} else if info.Mode().IsRegular() {
					out.Files++
					out.Bytes += info.Size()
				}
			}
			if readErr != nil {
				if readErr != io.EOF {
					out.Truncated = true
				}
				return
			}
		}
	}
	walk(".", 0)
	return out
}
func runtimeStorage(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), time.Second)
	defer cancel()
	all := func(string) bool { return true }
	root := sets.Path
	categories := map[string]storageCategory{}
	categories["preparation"] = storageSize(ctx, filepath.Join(root, "flow-pieces"), all)
	cachePath := ""
	if s := sets.BTsets(); s != nil && s.UseDisk {
		cachePath = s.TorrentsSavePath
	}
	categories["evicting_disk_cache"] = storageSize(ctx, cachePath, func(p string) bool { return pieceInventoryName.MatchString(p) })
	categories["diagnostics"] = storageSize(ctx, root, func(p string) bool {
		return strings.HasPrefix(p, "flow-incidents"+string(filepath.Separator)) || (filepath.Base(p) == p && strings.HasPrefix(p, "flow-history.jsonl"))
	}, func(p string) bool { return p == "flow-incidents" })
	categories["backups"] = storageSize(ctx, root, func(p string) bool {
		return filepath.Base(p) == p && (strings.Contains(p, "backup") || strings.HasSuffix(p, ".bak") || p == "flow-settings-good.json" || strings.HasPrefix(p, "flow-rejected-settings-"))
	}, func(string) bool { return false })
	c.Header("Cache-Control", "no-store")
	c.JSON(200, gin.H{"schema_version": 1, "preview_only": true, "categories": categories, "ram_cache_bytes": torrstor.Global().Allocations().RAMResidentBytes, "incidents": diagnostics.Incidents.Status(), "sampled_at": time.Now().UTC()})
}
