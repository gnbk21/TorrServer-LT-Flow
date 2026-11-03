package flow

import (
	"net/http"
	"testing"
)

func TestWholeShortVideosAreNotContainerProbes(t *testing.T) {
	for _, size := range []int64{MiB, 4 * MiB, 8 * MiB} {
		for _, header := range []string{"bytes=0-", "bytes=0-999999999"} {
			if got := Classify(http.MethodGet, false, ParseRangeHint(header, size), size, 0, false); got != "PLAYBACK" {
				t.Fatalf("whole %d-byte clip classified %s", size, got)
			}
		}
	}
	if got := Classify(http.MethodGet, false, ParseRangeHint("bytes=0-1023", MiB), MiB, 0, false); got != "HEAD_PROBE" {
		t.Fatal("partial header ceased to be a probe", got)
	}
	if got := Classify(http.MethodGet, false, ParseRangeHint("bytes=-65536", 100*MiB), 100*MiB, 0, false); got != "TAIL_INDEX" {
		t.Fatal("index ceased to be a probe", got)
	}
}

func TestRangeHintsAreNonAuthoritative(t *testing.T) {
	cases := []struct {
		header     string
		start, end int64
		valid      bool
	}{
		{"bytes=0-", 0, -1, true}, {"bytes=20-29", 20, 29, true},
		{"bytes=-10", 90, 99, true}, {"bytes=0-1,20-21", 0, -1, false},
		{"bytes=200-", 0, -1, false},
	}
	for _, c := range cases {
		got := ParseRangeHint(c.header, 100)
		if got.Start != c.start || got.End != c.end || got.Valid != c.valid {
			t.Fatalf("%q: %+v", c.header, got)
		}
	}
}
