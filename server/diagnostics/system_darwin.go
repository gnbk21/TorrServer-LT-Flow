//go:build darwin

package diagnostics

import "golang.org/x/sys/unix"

// No fabricated free-memory estimate on platforms without a qualified sample.
func systemMemory() (uint64, uint64, bool) {
	total, _ := unix.SysctlUint64("hw.memsize")
	return total, 0, false
}
func DiskFree(path string) (uint64, bool) {
	var s unix.Statfs_t
	err := unix.Statfs(path, &s)
	return uint64(s.Bavail) * uint64(s.Bsize), err == nil
}
