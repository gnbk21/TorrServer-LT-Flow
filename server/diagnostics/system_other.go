//go:build !windows && !linux && !android && !darwin

package diagnostics

func systemMemory() (uint64, uint64, bool) { return 0, 0, false }
func DiskFree(string) (uint64, bool)       { return 0, false }
