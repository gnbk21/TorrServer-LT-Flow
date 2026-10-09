package flow

import "testing"

func FuzzRangeHint(f *testing.F) {
	for _, header := range []string{"bytes=0-", "bytes=-1", "bytes=0-1,2-3", "bytes=9223372036854775807-", "bytes=-9223372036854775808"} {
		f.Add(header, int64(100))
	}
	f.Fuzz(func(t *testing.T, header string, size int64) {
		if len(header) > 8192 {
			return
		}
		hint := ParseRangeHint(header, size)
		if hint.Valid && (size <= 0 || hint.Start < 0 || hint.Start >= size || (hint.End != -1 && (hint.End < hint.Start || hint.End >= size))) {
			t.Fatal("out-of-file range", hint, size)
		}
		Classify("GET", false, hint, size, 0, false)
	})
}
