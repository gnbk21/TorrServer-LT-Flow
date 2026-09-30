package diagnostics

import "testing"

func TestProfilingRejectsPublicListeners(t *testing.T) {
	for _, address := range []string{"0.0.0.0:6060", "192.168.1.2:6060", "localhost:6060", "127.0.0.1:0", "[::]:6060"} {
		if stop, err := StartProfiling(address); err == nil {
			stop()
			t.Fatalf("accepted %s", address)
		}
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
