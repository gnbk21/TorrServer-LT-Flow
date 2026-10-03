//go:build !windows

package diagnostics

func protectFile(string) error { return nil }
