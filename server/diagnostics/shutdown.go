package diagnostics

import "sync/atomic"

var shutdownReason atomic.Pointer[string]

// Only enums are retained; API paths, signal text and error messages are not.
// Timeout is allowed to supersede a normal stop request.
func RequestShutdown(reason string) bool {
	switch reason {
	case "API_SHUTDOWN", "SIGNAL", "SERVICE_STOP", "SERVER_ERROR", "SHUTDOWN_TIMEOUT":
	default:
		return false
	}
	if reason == "SHUTDOWN_TIMEOUT" {
		shutdownReason.Store(&reason)
		return true
	}
	return shutdownReason.CompareAndSwap(nil, &reason)
}

func ShutdownReason() string {
	if value := shutdownReason.Load(); value != nil {
		return *value
	}
	return ""
}
