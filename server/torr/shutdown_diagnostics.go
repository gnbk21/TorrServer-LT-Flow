package torr

import (
	"server/diagnostics"
	"server/log"
	"time"
)

// The bounded sync is only used on exit paths, never during streaming.
func RecordShutdownReason(reason string) {
	if !diagnostics.RequestShutdown(reason) {
		return
	}
	if bt := helperEngine(); bt != nil {
		history := bt.history.Load()
		if !history.RecordExit(reason, 500*time.Millisecond) {
			log.Event("WARN", "Diagnostics", "Shutdown evidence could not be confirmed before its deadline.")
		}
	}
}
