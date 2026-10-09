package diagnostics

import (
	"net"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestProfilingRejectsPublicListeners(t *testing.T) {
	for _, address := range []string{"0.0.0.0:6060", "192.168.1.2:6060", "localhost:6060", "127.0.0.1:0", "[::]:6060"} {
		if stop, err := StartProfiling(address); err == nil {
			stop()
			t.Fatalf("accepted %s", address)
		}
	}
}

func TestDoctorDetectsInvalidAccountsAndOccupiedPort(t *testing.T) {
	dir := t.TempDir()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	port := strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)
	checks, ok := Doctor(dir, port, []string{"127.0.0.1"}, true)
	if ok {
		t.Fatal("occupied port and missing accounts accepted")
	}
	failures := 0
	for _, check := range checks {
		if check.Status == "error" {
			failures++
		}
	}
	if failures != 2 {
		t.Fatal(checks)
	}
	if err := os.WriteFile(filepath.Join(dir, "accs.db"), []byte(`{"fixture":"test-only"}`), 0600); err != nil {
		t.Fatal(err)
	}
	listener.Close()
	if checks, ok = Doctor(dir, port, []string{"127.0.0.1"}, true); !ok {
		t.Fatal(checks)
	}
	if _, ok := Doctor(dir, "65536", nil, false); ok {
		t.Fatal("invalid port accepted")
	}
}
func TestMemoryReportsGoAndNativeProcessSeparately(t *testing.T) {
	m := Memory()
	if m.GoHeapBytes == 0 || m.Goroutines == 0 || m.SampledAt.IsZero() {
		t.Fatal(m)
	}
	if m.RSSAvailable && m.RSSBytes == 0 {
		t.Fatal("RSS availability incorrect")
	}
}
