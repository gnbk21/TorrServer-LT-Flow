//go:build linux || android

package diagnostics

import (
	"golang.org/x/sys/unix"
	"os"
	"strconv"
	"strings"
)

func systemMemory() (uint64, uint64, bool) {
	b, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, 0, false
	}
	var total, available uint64
	for _, line := range strings.Split(string(b), "\n") {
		f := strings.Fields(line)
		if len(f) < 2 {
			continue
		}
		n, _ := strconv.ParseUint(f[1], 10, 64)
		if f[0] == "MemTotal:" {
			total = n * 1024
		}
		if f[0] == "MemAvailable:" {
			available = n * 1024
		}
	}
	return total, available, total > 0 && available > 0
}
func DiskFree(path string) (uint64, bool) {
	var s unix.Statfs_t
	err := unix.Statfs(path, &s)
	return uint64(s.Bavail) * uint64(s.Bsize), err == nil
}
