package log

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestRotationRetainsDescriptorAndBoundsArchives(t *testing.T) {
	path := filepath.Join(t.TempDir(), "server.log")
	f, err := openLogFile(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	r := &rotatingLog{file: f, path: path}
	block := bytes.Repeat([]byte("x"), int(LogFileBytes))
	for i := 0; i < 4; i++ {
		if _, err = r.Write(block); err != nil {
			t.Fatal(err)
		}
	}
	// Simulate a native writer retaining the original descriptor.
	if _, err = f.WriteString("native-tail"); err != nil {
		t.Fatal(err)
	}
	if _, err = r.Write([]byte("go-tail")); err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= LogArchives; i++ {
		info, err := os.Stat(fmt.Sprintf("%s.%d", path, i))
		if err != nil || info.Size() > LogFileBytes {
			t.Fatal("unbounded archive", info, err)
		}
	}
	if _, err = os.Stat(path + ".3"); !os.IsNotExist(err) {
		t.Fatal("unexpected archive")
	}
	data, _ := os.ReadFile(path)
	if !bytes.Contains(data, []byte("go-tail")) {
		t.Fatal("descriptor stopped working")
	}
}
