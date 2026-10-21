package flow

import "testing"

func TestErrorCodesRequireTheirCategory(t *testing.T) {
	for _, test := range []struct {
		category string
		code     int
		want     string
	}{
		{"HTTP", 503, "HTTP_5XX"}, {"SYSTEM", 503, "UNKNOWN"},
		{"DNS", 2, "DNS"}, {"ASIO", 2, "UNKNOWN"}, {"LIBTORRENT", 2, "UNKNOWN"},
		{"TLS", 2, "TLS"}, {"SYSTEM", 10060, "TIMEOUT"},
		{"GENERIC", 10061, "REFUSED"}, {"SYSTEM", 10051, "UNREACHABLE"},
		{"secret.example/token", 403, "UNKNOWN"},
	} {
		if got := NetworkErrorClass(test.category, test.code); got != test.want {
			t.Fatalf("%s/%d: %s, want %s", test.category, test.code, got, test.want)
		}
	}
}
