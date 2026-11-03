package torrstor

// CoverageRequirement reports existing protected header/index ranges only.
// It never creates demand, extends a lease or asks the native engine for data.
type CoverageRequirement struct {
	First, Last int
	Head, Index bool
}

func (c *Cache) CoverageRequirements() ([]CoverageRequirement, bool) {
	out := []CoverageRequirement{}
	truncated := false
	if c == nil {
		return out, false
	}
	add := func(r [2]int, head bool) {
		r[0], r[1] = max(0, r[0]), min(c.NumPieces-1, r[1])
		if r[1] < r[0] {
			return
		}
		row := CoverageRequirement{First: r[0], Last: r[1], Head: head, Index: !head}
		for _, existing := range out {
			if existing == row {
				return
			}
		}
		if len(out) == 8 {
			truncated = true
			return
		}
		out = append(out, row)
	}
	c.preloadMu.Lock()
	for i, r := range c.preloadProtect {
		add(r, i == 0)
	}
	c.preloadMu.Unlock()
	c.readersMu.Lock()
	for r := range c.readers {
		if r.internal {
			continue
		}
		add(r.headReserve(), true)
		add(r.tailReserve(), false)
		if truncated {
			break
		}
	}
	c.readersMu.Unlock()
	return out, truncated
}

// ResidentCoverage observes at most 256 pieces, without I/O, native calls,
// access-time changes, activation or reservations.
func (c *Cache) ResidentCoverage(indices []int) map[int][2]bool {
	out := make(map[int][2]bool)
	if c == nil {
		return out
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, index := range indices[:min(256, len(indices))] {
		if index < 0 || index >= c.NumPieces {
			continue
		}
		if piece := c.pieces[index]; piece != nil {
			piece.mu.RLock()
			out[index] = [2]bool{piece.size > 0, piece.complete && piece.size > 0}
			piece.mu.RUnlock()
		}
	}
	return out
}
