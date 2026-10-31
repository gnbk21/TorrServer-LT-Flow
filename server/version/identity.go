package version

import (
	"regexp"
	"runtime/debug"
)

type BuildIdentity struct {
	Version  string `json:"version"`
	Channel  string `json:"channel"`
	Source   string `json:"source"`
	Modified bool   `json:"modified"`
}

var stableBuildPattern = regexp.MustCompile(`^MatriX\.(?:145|148)\.Flow-v[0-9]+\.[0-9]+\.[0-9]+$`)
var previewBuildPattern = regexp.MustCompile(`^MatriX\.(?:145|148)\.Flow-(?:v[0-9]+\.[0-9]+\.[0-9]+-(?:alpha|beta|preview|rc)\.[0-9]+|preview\.[0-9]+)$`)

func Identity() BuildIdentity {
	i := BuildIdentity{Version: Version, Channel: "development"}
	if stableBuildPattern.MatchString(Version) {
		i.Channel = "stable"
	}
	if previewBuildPattern.MatchString(Version) {
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
