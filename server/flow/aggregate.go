package flow

import (
	"sync/atomic"
	"time"
)

// Process totals survive cache/torrent removal. They describe server-side
// observations, not decoded-player stalls, and reset only on process exit.
var processTotals struct {
	hits, misses, waits, waitNS, stalls, recoveries atomic.Uint64
	startup                                         [4]atomic.Uint64
	startupWaits                                    [len(startupWaitReasons)]atomic.Uint64
}

var startupWaitReasons = [...]string{"peer_discovery", "first_payload", "head_index", "server_prebuffer", "media_probe"}

func RecordStartupWait(stage string) {
	switch stage {
	case "PEERS":
		processTotals.startupWaits[0].Add(1)
	case "FIRST_BLOCK":
		processTotals.startupWaits[1].Add(1)
	case "HEAD_INDEX":
		processTotals.startupWaits[2].Add(1)
	case "PREBUFFER":
		processTotals.startupWaits[3].Add(1)
	case "PROBE_GRACE":
		processTotals.startupWaits[4].Add(1)
	}
}

type StartupWaitCount struct {
	Reason string
	Count  uint64
}

func StartupWaitCounts() [len(startupWaitReasons)]StartupWaitCount {
	var counts [len(startupWaitReasons)]StartupWaitCount
	for i, reason := range startupWaitReasons {
		counts[i] = StartupWaitCount{Reason: reason, Count: processTotals.startupWaits[i].Load()}
	}
	return counts
}

type ProcessTotals struct {
	HitBytes, MissBytes, Waits, WaitNS, Stalls, Recoveries uint64
	Startup                                                [4]uint64
}

func Totals() ProcessTotals {
	out := ProcessTotals{HitBytes: processTotals.hits.Load(), MissBytes: processTotals.misses.Load(), Waits: processTotals.waits.Load(), WaitNS: processTotals.waitNS.Load(), Stalls: processTotals.stalls.Load(), Recoveries: processTotals.recoveries.Load()}
	for i := range out.Startup {
		out.Startup[i] = processTotals.startup[i].Load()
	}
	return out
}
func RecordStartupOutcome(stage string) {
	var i int
	switch stage {
	case "READY":
		i = 0
	case "FAILED":
		i = 1
	case "TIMEOUT":
		i = 2
	case "CANCELLED":
		i = 3
	default:
		return
	}
	processTotals.startup[i].Add(1)
}
func RecordRecovery() { processTotals.recoveries.Add(1) }
func recordReadWait(d time.Duration) {
	processTotals.waits.Add(1)
	processTotals.waitNS.Add(uint64(d))
	if d >= 500*time.Millisecond {
		processTotals.stalls.Add(1)
	}
}
