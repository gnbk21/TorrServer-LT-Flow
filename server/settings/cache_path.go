package settings

import (
	"errors"
	"strings"
)

// Shell quoting is not part of a configured directory name. In particular,
// Windows' Copy as path includes quotes that make disk-cache writes fail.
// Keep inactive disk settings compatible and never silently rewrite a path.
func validateDiskCachePath(enabled bool, path string) error {
	path = strings.TrimSpace(path)
	if enabled && (strings.HasPrefix(path, `"`) || strings.HasSuffix(path, `"`)) {
		return errors.New("enter the disk cache folder path without surrounding quotation marks")
	}
	return nil
}
