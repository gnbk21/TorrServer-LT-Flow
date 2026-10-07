package torr

import (
	"net/http/httptest"
	"testing"
)

func TestProbeRequiresSecretAndDirectLoopback(t *testing.T) {
	r := httptest.NewRequest("GET", "/stream?stat=ffprobe", nil)
	r.RemoteAddr = "127.0.0.1:9876"
	if IsInternalProbe(r) {
		t.Fatal("query marker authorized probe")
	}
	q := r.URL.Query()
	q.Set("probe_key", probeAccessKey)
	r.URL.RawQuery = q.Encode()
	if !IsInternalProbe(r) {
		t.Fatal("internal probe rejected")
	}
	r.RemoteAddr = "192.0.2.1:9876"
	r.Header.Set("X-Forwarded-For", "127.0.0.1")
	if IsInternalProbe(r) {
		t.Fatal("forwarded header authorized remote probe")
	}
}
