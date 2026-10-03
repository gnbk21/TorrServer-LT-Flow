package flow

import (
	"errors"
	"io"
	"os"
	"path/filepath"
)

const MaxDHTStateBytes = 1 << 20

func ReadDHTFile(name string) ([]byte, error) {
	st, err := os.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !st.Mode().IsRegular() || st.Size() > MaxDHTStateBytes {
		return nil, errors.New("invalid DHT state file")
	}
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, MaxDHTStateBytes+1))
	if err == nil && len(b) > MaxDHTStateBytes {
		err = errors.New("DHT state exceeds limit")
	}
	return b, err
}

// WriteDHTFile atomically replaces one private, bounded native state file.
// A failed write leaves the previous usable state untouched.
func WriteDHTFile(name string, data []byte) error {
	if len(data) == 0 || len(data) > MaxDHTStateBytes {
		return errors.New("invalid DHT state size")
	}
	f, err := os.CreateTemp(filepath.Dir(name), ".flow-dht-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if _, err = f.Write(data); err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(tmp, name)
}
