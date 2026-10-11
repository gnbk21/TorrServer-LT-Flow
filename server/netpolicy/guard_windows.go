//go:build windows

package netpolicy

import (
	"os"
	"path/filepath"
)

func GuardSnapshot() GuardAvailability {
	status := GuardAvailability{Supported: true, State: "UNAVAILABLE"}
	executable, err := os.Executable()
	if err != nil {
		return status
	}
	return guardInDirectory(filepath.Dir(executable))
}

func guardInDirectory(directory string) GuardAvailability {
	status := GuardAvailability{Supported: true, State: "UNAVAILABLE"}
	path := filepath.Join(directory, "FlowNetworkGuard-windows-amd64.exe")
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() < 2 || info.Size() > 32<<20 {
		return status
	}
	file, err := os.Open(path)
	if err != nil {
		return status
	}
	defer file.Close()
	var magic [2]byte
	if n, err := file.Read(magic[:]); err != nil || n != 2 || magic != [2]byte{'M', 'Z'} {
		return status
	}
	status.Available = true
	status.State = "MANUAL_STATUS_REQUIRED"
	return status
}
