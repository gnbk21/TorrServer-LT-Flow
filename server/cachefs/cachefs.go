// Package cachefs confines cache operations beneath an owner-selected root.
// The root itself may be a deliberately selected directory alias. Descendant
// symlinks cannot escape it. This is not a sandbox against privileged mount
// changes or an administrator who deliberately selects a sensitive root.
package cachefs

import (
	"errors"
	"os"
	"path/filepath"
)

func Open(root string, create bool) (*os.Root, error) {
	if root == "" {
		return nil, errors.New("empty cache root")
	}
	if create {
		if err := os.MkdirAll(root, 0700); err != nil {
			return nil, err
		}
	}
	return os.OpenRoot(root)
}

func OpenFile(root, name string, flags int) (*os.File, error) {
	r, err := Open(root, flags&os.O_CREATE != 0)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return OpenIn(r, name, flags)
}

func OpenIn(r *os.Root, name string, flags int) (*os.File, error) {
	if flags&os.O_CREATE != 0 {
		if err := r.MkdirAll(filepath.Dir(name), 0700); err != nil {
			return nil, err
		}
	}
	// Reject known FIFOs/devices before opening, since even an attempted read
	// can block. The descriptor check below still handles replacement races.
	if st, err := r.Stat(name); err == nil {
		if !st.Mode().IsRegular() {
			return nil, errors.New("cache piece must be a regular file")
		}
	} else if !os.IsNotExist(err) || flags&os.O_CREATE == 0 {
		return nil, err
	}
	// Inspect the actual descriptor before any caller can truncate a hard link.
	f, err := r.OpenFile(name, flags&^os.O_TRUNC, 0600)
	if err != nil {
		return nil, err
	}
	st, err := f.Stat()
	if err == nil && (!st.Mode().IsRegular() || !singleLink(f, st)) {
		err = errors.New("cache piece must be a regular file with one link")
	}
	if err != nil {
		f.Close()
		return nil, err
	}
	if flags&os.O_TRUNC != 0 {
		if err := f.Truncate(0); err != nil {
			f.Close()
			return nil, err
		}
	}
	return f, nil
}

func Stat(root, name string) (os.FileInfo, error) {
	f, err := OpenFile(root, name, os.O_RDONLY)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return f.Stat()
}

func MkdirAll(root, name string) error {
	r, err := Open(root, true)
	if err != nil {
		return err
	}
	defer r.Close()
	return r.MkdirAll(name, 0700)
}

func Remove(root, name string) error {
	r, err := Open(root, false)
	if err != nil {
		return err
	}
	defer r.Close()
	return r.Remove(name)
}

func RemoveAll(root, name string) error {
	r, err := Open(root, false)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer r.Close()
	return r.RemoveAll(name)
}

// RemoveFlat keeps legacy nonrecursive cleanup behavior. A malicious child
// directory alias cannot redirect enumeration/removal outside the opened root.
func RemoveFlat(root *os.Root, name string) error {
	dir, err := root.OpenRoot(name)
	if err != nil {
		return err
	}
	defer dir.Close()
	f, err := dir.Open(".")
	if err != nil {
		return err
	}
	entries, err := f.ReadDir(-1)
	f.Close()
	if err != nil {
		return err
	}
	for _, e := range entries {
		if err := dir.Remove(e.Name()); err != nil {
			return err
		}
	}
	dir.Close()
	return root.Remove(name)
}
