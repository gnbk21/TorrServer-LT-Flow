package api

import (
	"context"
	"github.com/gin-gonic/gin"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
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

// Walk only configured runtime roots, never the developer workspace or arbitrary
// caller paths. Symlinks/junctions and nonregular files are not followed.
func storageSize(ctx context.Context, root string, accept func(string) bool, descend ...func(string) bool) storageCategory {
	out := storageCategory{Protected: true}
	if root == "" {
		return out
	}
	st, err := os.Lstat(root)
	if os.IsNotExist(err) {
		return out
	}
	if err != nil || !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
		out.Truncated = true
		return out
	}
	visited := 0
	_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		visited++
		if ctx.Err() != nil || visited > 10000 {
			out.Truncated = true
			return fs.SkipAll
		}
		if err != nil {
			out.Truncated = true
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			out.Truncated = true
			return nil
		}
		if entry.IsDir() {
			if path != root && len(descend) > 0 {
				rel, err := filepath.Rel(root, path)
				if err != nil || !descend[0](rel) {
					return fs.SkipDir
				}
			}
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil || !accept(rel) {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			out.Truncated = true
			return nil
		}
		if info.Mode().IsRegular() {
			out.Files++
			out.Bytes += info.Size()
		}
		return nil
	})
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
