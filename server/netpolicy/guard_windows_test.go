//go:build windows

package netpolicy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGuardPresenceNeverClaimsInstalledOrVerified(t *testing.T) {
	dir := t.TempDir()
	status := guardInDirectory(dir)
	if !status.Supported || status.Available || status.State != "UNAVAILABLE" {
		t.Fatal(status)
	}
	path := filepath.Join(dir, "FlowNetworkGuard-windows-amd64.exe")
	if err := os.WriteFile(path, []byte("invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	if guardInDirectory(dir).Available {
		t.Fatal("non-PE file was treated as a helper")
	}
	if err := os.WriteFile(path, []byte("MZ fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	status = guardInDirectory(dir)
	if !status.Available || status.State != "MANUAL_STATUS_REQUIRED" || status.DNSVerified || status.LeakProtectionVerified {
		t.Fatal("presence was confused with installed or verified protection", status)
	}
}
