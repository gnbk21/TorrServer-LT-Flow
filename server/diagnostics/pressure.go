package diagnostics

import "time"

// pressurePolicy consumes only fresh OS samples. Repeated callers of Memory's
// five-second cache cannot advance recovery or cause a budget oscillation.
type pressurePolicy struct {
	active     bool
	recovering int
	last       time.Time
}

func pressureEntry(total uint64) uint64 { return max(uint64(512<<20), total/20) }
func (p *pressurePolicy) sample(total, available uint64, known bool, at time.Time) {
	if !p.last.IsZero() && at.Sub(p.last) < 5*time.Second {
		return
	}
	p.last = at
	if !known {
		p.recovering = 0
		return
	}
	threshold := pressureEntry(total)
	if available < threshold {
		p.active = true
		p.recovering = 0
		return
	}
	if !p.active {
		return
	}
	if available <= threshold+threshold/2 {
		p.recovering = 0
		return
	}
	p.recovering++
	if p.recovering == 3 {
		p.active = false
		p.recovering = 0
	}
}
func (p *pressurePolicy) state(known bool) string {
	if p.active {
		if p.recovering > 0 {
			return "recovering"
		}
		return "pressure"
	}
	if !known {
		return "unknown"
	}
	return "normal"
}
