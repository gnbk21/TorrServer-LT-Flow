package torr

import storageState "server/torr/storage/state"

const stragglerEscalatePieces = 8

func preloadGateTotals(snapshot map[int]storageState.ItemState, pieces []int, pieceLength int64) (got int64, done int) {
	for _, piece := range pieces {
		item := snapshot[piece]
		if item.Completed {
			done++
			got += pieceLength
		} else {
			got += min(pieceLength, max(int64(0), item.Size))
		}
	}
	return
}

// Recovery observes the current phase, including pieces already deadlined.
// Tail-first ordering matches bootstrap; no future phase gains demand here.
func preloadRecoveryPieces(snapshot map[int]storageState.ItemState, active []int, tailFirst, tailLast int) []int {
	var result []int
	for _, tail := range []bool{true, false} {
		for _, piece := range active {
			if (piece >= tailFirst && piece <= tailLast) == tail && !snapshot[piece].Completed {
				result = append(result, piece)
				if len(result) == stragglerEscalatePieces {
					return result
				}
			}
		}
	}
	return result
}

// Only the current phase may create urgent demand. In particular, an unfinished
// bootstrap must not promote the phase-2 head just because the entire gate has
// few large pieces. Existing deadlines retain their original order and age.
func preloadStragglers(snapshot map[int]storageState.ItemState, active []int, tailFirst, tailLast int, deadlined map[int]bool) []int {
	missing := 0
	for _, p := range active {
		if !snapshot[p].Completed {
			missing++
		}
	}
	if missing == 0 || missing > stragglerEscalatePieces {
		return nil
	}
	var result []int
	for _, tail := range []bool{true, false} {
		for _, p := range active {
			if (p >= tailFirst && p <= tailLast) != tail || snapshot[p].Completed || deadlined[p] {
				continue
			}
			result = append(result, p)
		}
	}
	return result
}
