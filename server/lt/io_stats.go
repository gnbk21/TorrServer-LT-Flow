package lt

/*
#include "lt_disk_io.h"
*/
import "C"

// StorageIO is process-wide. Latency buckets are upper bounds, not exact
// percentiles, and include reads, writes, hashing and maintenance fences.
type StorageIO struct {
	QueuedBytes    uint64 `json:"queued_bytes"`
	PeakQueueBytes uint64 `json:"peak_queue_bytes"`
	QueuedJobs     uint64 `json:"queued_jobs"`
	RejectedJobs   uint64 `json:"rejected_jobs"`
	CompletedJobs  uint64 `json:"completed_jobs"`
	WaitP95Us      uint64 `json:"wait_p95_us"`
	WaitMaxUs      uint64 `json:"wait_max_us"`
	CallbackP95Us  uint64 `json:"callback_p95_us"`
	CallbackMaxUs  uint64 `json:"callback_max_us"`
}

func StorageIOStats() StorageIO {
	var s C.struct_tsl_io_stats
	C.lt_storage_io_stats(&s)
	return StorageIO{uint64(s.queued_bytes), uint64(s.peak_queue_bytes), uint64(s.queued_jobs),
		uint64(s.rejected_jobs), uint64(s.completed_jobs), uint64(s.wait_p95_us),
		uint64(s.wait_max_us), uint64(s.callback_p95_us), uint64(s.callback_max_us)}
}
