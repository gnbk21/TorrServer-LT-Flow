package settings

import "testing"

func TestDiskCachePathRejectsShellQuotes(t *testing.T) {
	for _, path := range []string{`"C:\cache folder"`, `"C:\cache`, `C:\cache"`, `  "C:\cache"  `, `"/tmp/cache"`} {
		if validateDiskCachePath(true, path) == nil {
			t.Errorf("accepted shell-quoted disk path %q", path)
		}
		if err := validateDiskCachePath(false, path); err != nil {
			t.Errorf("rejected an inactive disk path: %v", err)
		}
	}
	for _, path := range []string{`C:\cache folder`, `\\server\share\cache`, `/tmp/cache`, `relative cache`, ``} {
		if err := validateDiskCachePath(true, path); err != nil {
			t.Errorf("changed existing unquoted path behavior for %q: %v", path, err)
		}
	}
}
