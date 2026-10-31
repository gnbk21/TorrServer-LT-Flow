package api

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestStorageInventoryIsBoundedReadOnlyAndDoesNotFollowLinks(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "owned"), []byte("123"), 0600); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "private"), []byte("outside bytes must not be counted"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "outside")); err == nil {
		out := storageSize(context.Background(), root, func(string) bool { return true })
		if !out.Truncated || out.Bytes != 3 || out.Files != 1 {
			t.Fatal(out)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if !storageSize(ctx, root, func(string) bool { return true }).Truncated {
		t.Fatal("cancel ignored")
	}
	if st, err := os.Stat(filepath.Join(root, "owned")); err != nil || st.Size() != 3 {
		t.Fatal("inventory modified data")
	}
}

func TestStorageInventoryPinsSelectedDirectory(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "cache")
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "owned"), []byte("123"), 0600); err != nil {
		t.Fatal(err)
	}
	replaced := false
	out := storageSize(context.Background(), root, func(string) bool {
		if replaced {
			return true
		}
		replaced = true
		if err := os.Rename(root, root+"-moved"); err != nil {
			t.Skip("platform denies renaming an open directory:", err)
		}
		if err := os.Mkdir(root, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "owned"), []byte("replacement bytes must not be counted"), 0600); err != nil {
			t.Fatal(err)
		}
		return true
	})
	if !replaced || out.Truncated || out.Files != 1 || out.Bytes != 3 {
		t.Fatal("inventory followed a replacement at the selected pathname", out)
	}
}

func TestStorageInventoryPrunesUnrelatedTrees(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"flow-pieces/nested", "flow-incidents"} {
		if err := os.MkdirAll(filepath.Join(root, name), 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "flow-incidents", "incident.trace"), []byte("123"), 0600); err != nil {
		t.Fatal(err)
	}
	out := storageSize(context.Background(), root, func(string) bool { return true }, func(p string) bool { return p == "flow-incidents" })
	if out.Files != 1 || out.Bytes != 3 || out.Truncated {
		t.Fatal(out)
	}
}
