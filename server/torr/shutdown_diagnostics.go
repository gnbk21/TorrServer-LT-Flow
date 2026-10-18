package torr

import (
	"server/diagnostics"
	"server/flow"
	"time"
)

// The bounded sync is only used on exit paths, never during streaming.
func RecordShutdownReason(reason string) {
	if !diagnostics.RequestShutdown(reason) {
		return
	}
	if bt := helperEngine(); bt != nil {
		history := bt.history.Load()
		history.Record(flow.HistoryEvent{Type: "shutdown_requested", Stage: reason})
		history.Flush(500 * time.Millisecond)
	}
}
