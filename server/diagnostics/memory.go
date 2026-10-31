package diagnostics

import (
	"runtime"
	"sync"
	"time"
)

type MemoryStatus struct {
	RSSBytes             uint64    `json:"rss_bytes"`
	RSSAvailable         bool      `json:"rss_available"`
	Handles              uint64    `json:"handles"`
	HandlesAvailable     bool      `json:"handles_available"`
	GoHeapBytes          uint64    `json:"go_heap_bytes"`
	GoSystemBytes        uint64    `json:"go_system_bytes"`
	Goroutines           int       `json:"goroutines"`
	SampledAt            time.Time `json:"sampled_at"`
	SystemTotalBytes     uint64    `json:"system_total_bytes"`
	SystemAvailableBytes uint64    `json:"system_available_bytes"`
	SystemAvailable      bool      `json:"system_available"`
	Pressure             bool      `json:"pressure"`
	PressureState        string    `json:"pressure_state"`
	RecoverySamples      int       `json:"recovery_samples"`
	PressureEntryBytes   uint64    `json:"pressure_entry_bytes"`
	PressureExitBytes    uint64    `json:"pressure_exit_bytes"`
}

var memoryMu sync.Mutex
var memorySample MemoryStatus
var memoryPressure pressurePolicy

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
	memorySample = MemoryStatus{RSSBytes: rss, RSSAvailable: rssOK, Handles: handles, HandlesAvailable: handlesOK, GoHeapBytes: m.HeapAlloc, GoSystemBytes: m.Sys, Goroutines: runtime.NumGoroutine(), SampledAt: time.Now()}
	total, available, known := systemMemory()
	memorySample.SystemTotalBytes, memorySample.SystemAvailableBytes, memorySample.SystemAvailable = total, available, known
	memoryPressure.sample(total, available, known, memorySample.SampledAt)
	memorySample.Pressure = memoryPressure.active
	memorySample.PressureState = memoryPressure.state(known)
	memorySample.RecoverySamples = memoryPressure.recovering
	memorySample.PressureEntryBytes = pressureEntry(total)
	memorySample.PressureExitBytes = pressureEntry(total) + pressureEntry(total)/2
	return memorySample
}
