package ffprobe

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"gopkg.in/vansante/go-ffprobe.v2"
)

// binFile is the resolved ffprobe path, empty when none was found.
var binFile string

func init() {
	// PATH first.
	if path, err := exec.LookPath("ffprobe"); err == nil {
		ffprobe.SetFFProbeBinPath(path)
		binFile = path
		return
	}
	// Fallback: a binary shipped next to the server's own executable. The old
	// fallback kept a bare "ffprobe" when one sat in the working directory:
	// Exists() then reported true, but exec never searches the working
	// directory, so every probe failed.
	fallback := filepath.Join(filepath.Dir(os.Args[0]), "ffprobe")
	if runtime.GOOS == "windows" {
		fallback += ".exe"
	}
	if _, err := os.Stat(fallback); err == nil {
		ffprobe.SetFFProbeBinPath(fallback)
		binFile = fallback
	}
}

func Exists() bool {
	if binFile == "" {
		return false
	}
	_, err := os.Stat(binFile)
	return err == nil
}

// ProbeUrl probes the media at link. extra passes additional ffprobe input
// options (placed before the URL by the library), e.g. -analyzeduration /
// -probesize caps to keep a probe near the file head instead of reading deep
// into — or seeking to the end of — the file.
func ProbeUrl(link string, extra ...string) (*ffprobe.ProbeData, error) {
	return ProbeURLContext(context.Background(), link, extra...)
}

func ProbeURLContext(parent context.Context, link string, extra ...string) (*ffprobe.ProbeData, error) {
	ctx, cancel := context.WithTimeout(parent, 5*time.Minute)
	defer cancel()
	return ffprobe.ProbeURL(ctx, link, extra...)
}

func ProbeReader(reader io.Reader) (*ffprobe.ProbeData, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	return ffprobe.ProbeReader(ctx, reader)
}
