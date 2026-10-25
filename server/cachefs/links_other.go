//go:build !windows && !linux && !darwin && !android && !freebsd

package cachefs

import "os"

// Unsupported targets fail closed rather than assume link ownership.
func singleLink(*os.File, os.FileInfo) bool { return false }
