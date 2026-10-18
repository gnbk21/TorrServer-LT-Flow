package diagnostics

import "testing"

func TestShutdownReasonAllowlistAndTimeoutPrecedence(t *testing.T) {
	old := shutdownReason.Load()
	shutdownReason.Store(nil)
	t.Cleanup(func() { shutdownReason.Store(old) })
	if RequestShutdown("/shutdown/private?token=secret") || ShutdownReason() != "" {
		t.Fatal("untrusted reason accepted")
	}
	if !RequestShutdown("SIGNAL") || RequestShutdown("API_SHUTDOWN") || ShutdownReason() != "SIGNAL" {
		t.Fatal("first cause overwritten")
	}
	if !RequestShutdown("SHUTDOWN_TIMEOUT") || ShutdownReason() != "SHUTDOWN_TIMEOUT" {
		t.Fatal("timeout hidden")
	}
}
