package cachefs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfinedCacheIOAndLinks(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	secret := filepath.Join(outside, "secret")
	if err := os.WriteFile(secret, []byte("preserved"), 0600); err != nil {
		t.Fatal(err)
	}
	f, err := OpenFile(root, "hash/0", os.O_CREATE|os.O_RDWR)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write([]byte("piece")); err != nil {
		t.Fatal(err)
	}
	f.Close()
	for _, name := range []string{"../secret", secret} {
		if f, err := OpenFile(root, name, os.O_RDWR|os.O_TRUNC); err == nil {
			f.Close()
			t.Fatal("escaped root", name)
		}
	}
	if err := os.Link(secret, filepath.Join(root, "hardlink")); err != nil {
		t.Fatal(err)
	}
	if f, err := OpenFile(root, "hardlink", os.O_RDWR|os.O_TRUNC); err == nil {
		f.Close()
		t.Fatal("hardlink was opened")
	}
	if b, err := os.ReadFile(secret); err != nil || string(b) != "preserved" {
		t.Fatal("hardlink target changed", err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "alias")); err != nil {
		t.Skip("symlink creation unavailable:", err)
	}
	for _, flags := range []int{os.O_RDONLY, os.O_CREATE | os.O_RDWR} {
		if f, err := OpenFile(root, "alias/secret", flags); err == nil {
			f.Close()
			t.Fatal("followed escaping directory")
		}
	}
	if err := Remove(root, "alias/secret"); err == nil {
		t.Fatal("removed outside file")
	}
	r, err := Open(root, false)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if err := RemoveFlat(r, "alias"); err == nil {
		t.Fatal("enumerated escaping directory")
	}
	if err := RemoveAll(root, "alias"); err != nil {
		t.Fatal(err)
	}
	if b, err := os.ReadFile(secret); err != nil || string(b) != "preserved" {
		t.Fatal("outside file changed", err)
	}
	if err := RemoveFlat(r, "hash"); err != nil {
		t.Fatal(err)
	}
}

func TestSelectedRootAliasAndPinnedHandle(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "cache")
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(parent, "selected-alias")
	if err := os.Symlink(root, alias); err != nil {
		t.Skip("symlink creation unavailable:", err)
	}
	f, err := OpenFile(alias, "hash/0", os.O_CREATE|os.O_RDWR)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	r, err := Open(root, false)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	// A replacement at the original pathname must not change the opened root.
	if err := os.Rename(root, root+"-moved"); err != nil {
		t.Skip("platform denies renaming an open root:", err)
	}
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Stat("hash/0"); err != nil {
		t.Fatal("opened root followed replacement", err)
	}
	if _, err := os.Stat(filepath.Join(root, "hash", "0")); !os.IsNotExist(err) {
		t.Fatal("replacement root unexpectedly used")
	}
}
