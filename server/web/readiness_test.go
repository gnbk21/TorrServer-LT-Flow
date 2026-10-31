package web

import "testing"

func TestSetupAvailabilityDoesNotAdmitEngineOperations(t *testing.T) {
	for _, path := range []string{"/", "/assets/app.js", "/echo", "/flow/network", "/settings/metadata/v1", "/flow/pairing", "/flow/pairing/status/id", "/flow/pairing/proof/token"} {
		if !engineIndependentPath(path) {
			t.Fatal("setup blocked while starting", path)
		}
	}
	for _, path := range []string{"/settings", "/flow/control", "/flow/preparation", "/stream", "/play/hash/1", "/flow/pairing-other", "/shutdown"} {
		if engineIndependentPath(path) {
			t.Fatal("engine operation admitted while starting", path)
		}
	}
}
