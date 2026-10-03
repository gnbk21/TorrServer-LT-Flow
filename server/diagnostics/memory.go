package diagnostics

import (
	"runtime"
	"sync"
	"time"
)

type MemoryStatus struct {
	RSSBytes         uint64    `json:"rss_bytes"`
	RSSAvailable     bool      `json:"rss_available"`
	Handles          uint64    `json:"handles"`
	HandlesAvailable bool      `json:"handles_available"`
	GoHeapBytes      uint64    `json:"go_heap_bytes"`
	GoSystemBytes    uint64    `json:"go_system_bytes"`
	Goroutines       int       `json:"goroutines"`
	SampledAt        time.Time `json:"sampled_at"`
}

var memoryMu sync.Mutex
var memorySample MemoryStatus

// Cache the OS query for five seconds. RSS includes native allocations and
// resident code; subtracting the Go heap does not measure libtorrent's heap.
func Memory() MemoryStatus {
	memoryMu.Lock()
	defer memoryMu.Unlock()
	if time.Since(memorySample.SampledAt) < 5*time.Second {
		return memorySample
	}
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	rss, handles, rssOK, handlesOK := processMemory()
	memorySample = MemoryStatus{rss, rssOK, handles, handlesOK, m.HeapAlloc, m.Sys, runtime.NumGoroutine(), time.Now()}
	return memorySample
}
