package server

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"server/settings"
)

func TestCleanCachePreservesNonHashOwnerDirectories(t *testing.T) {
	root := t.TempDir()
	previous := settings.BTsets()
	settings.StoreBTsets(&settings.BTSets{UseDisk: true, TorrentsSavePath: root, RemoveCacheOnDrop: true})
	t.Cleanup(func() { settings.StoreBTsets(previous) })
	for _, name := range []string{strings.Repeat("z", 40), strings.Repeat("z", 64), strings.Repeat("a", 40), strings.Repeat("b", 64)} {
		dir := filepath.Join(root, name)
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "0"), []byte("owned"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	cleanCache()
	for _, name := range []string{strings.Repeat("z", 40), strings.Repeat("z", 64)} {
		if _, err := os.Stat(filepath.Join(root, name, "0")); err != nil {
			t.Fatalf("non-hash directory was removed: %v", err)
		}
	}
	for _, name := range []string{strings.Repeat("a", 40), strings.Repeat("b", 64)} {
		if _, err := os.Stat(filepath.Join(root, name)); !os.IsNotExist(err) {
			t.Fatalf("cache directory was not removed: %v", err)
		}
	}
}
