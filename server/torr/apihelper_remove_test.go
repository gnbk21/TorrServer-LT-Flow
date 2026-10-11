package torr

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRemTorrentRejectsUntrustedCacheComponents(t *testing.T) {
	root := t.TempDir()
	sentinel := filepath.Join(root, "keep")
	if err := os.WriteFile(sentinel, []byte("retained"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, input := range []string{"", "/", "..", "../keep", sentinel,
		strings.Repeat("z", 40), strings.Repeat("z", 64),
		strings.Repeat("a", 38) + "..", strings.Repeat("b", 62) + "/x"} {
		if err := RemTorrent(input); err == nil || !strings.Contains(err.Error(), "info hash:") {
			t.Fatalf("unvalidated removal %q: %v", input, err)
		}
	}
	if got, err := os.ReadFile(sentinel); err != nil || string(got) != "retained" {
		t.Fatalf("owned sentinel was altered: %q %v", got, err)
	}
}
