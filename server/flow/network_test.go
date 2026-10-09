package flow

import (
	"net"
	"testing"
	"time"
)

func TestRetryDelayIsBounded(t *testing.T) {
	want := []time.Duration{2, 4, 8, 16, 32, 60, 60}
	for i, seconds := range want {
		if got := RetryDelay(i, 2, 60); got != seconds*time.Second {
			t.Fatalf("attempt %d: got %v, want %vs", i, got, seconds)
		}
	}
}

func TestUsableLocalIP(t *testing.T) {
	for address, want := range map[string]bool{
		"127.0.0.1": false, "0.0.0.0": false, "169.254.1.1": false,
		"::1": false, "fe80::1": false, "192.168.1.20": true,
		"2001:db8::1": true,
	} {
		if got := UsableLocalIP(net.ParseIP(address)); got != want {
			t.Fatalf("%s: got %v, want %v", address, got, want)
		}
	}
}
