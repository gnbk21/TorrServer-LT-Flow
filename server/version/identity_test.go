package version

import "testing"

func TestUnstampedBuildCannotPretendToBeStable(t *testing.T) {
	old := Version
	defer func() { Version = old }()
	for _, test := range []struct{ version, channel string }{
		{"MatriX.145.Flow-dev-local", "development"},
		{"MatriX.145.Flow-v1.1.3", "stable"},
		{"MatriX.145.Flow-v1.2.0-preview.1", "preview"},
	} {
		Version = test.version
		if got := Identity(); got.Channel != test.channel || got.Version != test.version {
			t.Fatal(got)
		}
	}
}
