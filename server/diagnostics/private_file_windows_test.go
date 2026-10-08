//go:build windows

package diagnostics

import (
	"golang.org/x/sys/windows"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrivateFileHasProtectedWindowsACL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "generated.key")
	if err := AtomicPrivateFile(path, []byte("fixture-secret")); err != nil {
		t.Fatal(err)
	}
	if err := RestrictPrivateFile(path); err != nil {
		t.Fatal(err)
	}
	descriptor, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		t.Fatal(err)
	}
	sddl := descriptor.String()
	if !strings.Contains(sddl, "D:P") || strings.Contains(sddl, ";;;WD)") || strings.Contains(sddl, ";;;BU)") || strings.Count(sddl, "(A;") != 3 {
		t.Fatal("private file ACL is not restricted to process account, administrators and SYSTEM")
	}
	if data, err := os.ReadFile(path); err != nil || string(data) != "fixture-secret" {
		t.Fatal("protected contents changed")
	}
}
