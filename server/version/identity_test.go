package version

import "testing"

func TestIdentityReleaseChannels(t *testing.T) {
	old := Version
	defer func() { Version = old }()
	for _, row := range []struct{ version, channel string }{{"MatriX.145.Flow-v1.1.5", "stable"}, {"MatriX.145.Flow-v1.2.0-alpha.2", "preview"}, {"MatriX.145.Flow-v1.2.0-beta.3", "preview"}, {"MatriX.145.Flow-v1.2.0-rc.1", "preview"}, {"MatriX.145.Flow-preview.7", "preview"}, {"MatriX.145.Flow-dev-unknown", "development"}} {
		Version = row.version
		if Identity().Channel != row.channel {
			t.Fatalf("wrong channel: %s", Version)
		}
	}
}

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
