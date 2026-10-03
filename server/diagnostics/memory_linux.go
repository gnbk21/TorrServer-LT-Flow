//go:build linux || android

package diagnostics

import (
	"fmt"
	"os"
)

func processMemory() (uint64, uint64, bool, bool) {
	buf, err := os.ReadFile("/proc/self/statm")
	var size, resident uint64
	if err != nil {
		return 0, 0, false, false
	}
	if _, err = fmt.Sscan(string(buf), &size, &resident); err != nil {
		return 0, 0, false, false
	}
	fds, fdErr := os.ReadDir("/proc/self/fd")
	return resident * uint64(os.Getpagesize()), uint64(len(fds)), true, fdErr == nil
}
