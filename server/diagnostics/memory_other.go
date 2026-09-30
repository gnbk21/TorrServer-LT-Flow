//go:build !windows && !linux && !android

package diagnostics

import (
	"context"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func processMemory() (uint64, uint64, bool, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "ps", "-o", "rss=", "-p", strconv.Itoa(os.Getpid())).Output()
	if err != nil {
		return 0, 0, false, false
	}
	rss, err := strconv.ParseUint(strings.TrimSpace(string(out)), 10, 64)
	return rss * 1024, 0, err == nil, false
}
