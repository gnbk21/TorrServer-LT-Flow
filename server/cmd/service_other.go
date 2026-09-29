//go:build !windows

package main

import "errors"

func serviceDataDir() string { return "" }

func serviceCommand(_ string, _ *args) error {
	return errors.New("Windows services are supported on Windows only")
}

func runWindowsService() error { return errors.New("Windows services are supported on Windows only") }
