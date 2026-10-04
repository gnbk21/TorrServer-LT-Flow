package torr

import (
	"server/settings"
	"server/torr/utils"
)

// publicTrackerTiers preserves configured tier order and removes duplicates.
// Call only after native metadata establishes that the torrent is public.
func publicTrackerTiers(original [][]string) [][]string {
	tiers := original
	if s := settings.BTsets(); s != nil {
		switch s.RetrackersMode {
		case 1:
			tiers = append(append([][]string(nil), original...), utils.GetDefTrackers())
		case 2:
			tiers = nil
		case 3:
			tiers = [][]string{utils.GetDefTrackers()}
		}
	}
	tiers = append(append([][]string(nil), tiers...), utils.GetTrackerFromFile())
	seen := make(map[string]bool)
	out := make([][]string, 0, len(tiers))
	for _, tier := range tiers {
		var urls []string
		for _, url := range tier {
			if url != "" && !seen[url] {
				seen[url] = true
				urls = append(urls, url)
			}
		}
		if len(urls) > 0 {
			out = append(out, urls)
		}
	}
	return out
}
