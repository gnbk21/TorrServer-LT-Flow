package console

import (
	"os"

	"golang.org/x/sys/windows"
)

// ConfigureColor restores the console mode on normal shutdown. Pipes, old
// consoles, NO_COLOR and TERM=dumb always receive plain text.
func ConfigureColor(f *os.File, mode string) (bool, func()) {
	noop := func() {}
	if mode != "auto" || os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return false, noop
	}
	h := windows.Handle(f.Fd())
	var original uint32
	if windows.GetConsoleMode(h, &original) != nil {
		return false, noop
	}
	next := original | windows.ENABLE_PROCESSED_OUTPUT | windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING
	if windows.SetConsoleMode(h, next) != nil {
		return false, noop
	}
	return true, func() { _ = windows.SetConsoleMode(h, original) }
}
