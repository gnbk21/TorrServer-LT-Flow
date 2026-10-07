package diagnostics

import (
	"os"
	"path/filepath"
)

// AtomicPrivateFile restricts the temporary file before any credentials are
// written, syncs it, then replaces the destination on the same filesystem.
func AtomicPrivateFile(path string, data []byte) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".flow-private-*")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if err = protectFile(name); err == nil {
		_, err = f.Write(data)
	}
	if err == nil {
		err = f.Sync()
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(name, path)
}

// A recovery backup can include local credentials. Protect it before writing,
// unlike redacted portable exports and support reports.
func WritePrivateFile(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	ok := false
	defer func() {
		f.Close()
		if !ok {
			os.Remove(path)
		}
	}()
	if err = protectFile(path); err != nil {
		return err
	}
	if _, err = f.Write(data); err != nil {
		return err
	}
	if err = f.Sync(); err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	ok = true
	return nil
}
