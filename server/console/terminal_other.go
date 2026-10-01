//go:build !windows

package console

import (
	"os"

	"github.com/mattn/go-isatty"
)

func ConfigureColor(f *os.File, mode string) (bool, func()) {
	return mode == "auto" && os.Getenv("NO_COLOR") == "" && os.Getenv("TERM") != "dumb" && isatty.IsTerminal(f.Fd()), func() {}
}
