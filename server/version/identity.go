package version

import (
	"runtime/debug"
	"strings"
)

type BuildIdentity struct {
	Version  string `json:"version"`
	Channel  string `json:"channel"`
	Source   string `json:"source"`
	Modified bool   `json:"modified"`
}

func Identity() BuildIdentity {
	i := BuildIdentity{Version: Version, Channel: "development"}
	if strings.Contains(Version, ".Flow-v") && !strings.Contains(Version, "preview") && !strings.Contains(Version, "-rc") {
		i.Channel = "stable"
	}
	if strings.Contains(Version, "preview") || strings.Contains(Version, "-rc") {
		i.Channel = "preview"
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				i.Source = setting.Value
			}
			if setting.Key == "vcs.modified" {
				i.Modified = setting.Value == "true"
			}
		}
	}
	return i
}
