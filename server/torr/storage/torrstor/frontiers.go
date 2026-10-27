package torrstor

import "time"

const recoveryFrontierLimit = 8
const recoveryFrontierTTL = 3 * time.Second

// Each scheduling owner has its own lease. Refreshing preload must not revive
// an obsolete reader miss, and retiring preload must not erase a live reader.
// Cache.frontierMu protects this state and its native publication together.
type recoveryFrontiers struct {
	readers, preload     []int
	readersAt, preloadAt time.Time
}

func (f *recoveryFrontiers) update(pieces []int, preload bool, now time.Time, numPieces int) []int {
	selected := make([]int, 0, recoveryFrontierLimit)
	appendPieces := func(pieces []int) {
		for _, piece := range pieces {
			if piece < 0 || piece >= numPieces || len(selected) >= recoveryFrontierLimit {
				continue
			}
			duplicate := false
			for _, previous := range selected {
				if piece == previous {
					duplicate = true
					break
				}
			}
			if !duplicate {
				selected = append(selected, piece)
			}
		}
	}
	appendPieces(pieces)
	if preload {
		f.preload, f.preloadAt = selected, now
	} else {
		f.readers, f.readersAt = selected, now
	}
	selected = make([]int, 0, recoveryFrontierLimit)
	if !now.Before(f.readersAt) && now.Sub(f.readersAt) <= recoveryFrontierTTL {
		appendPieces(f.readers)
	}
	if !now.Before(f.preloadAt) && now.Sub(f.preloadAt) <= recoveryFrontierTTL {
		appendPieces(f.preload)
	}
	return selected
}
