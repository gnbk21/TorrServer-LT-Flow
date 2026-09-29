package flow

import "testing"

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
