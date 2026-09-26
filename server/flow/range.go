package flow

import (
	"net/http"
	"strconv"
	"strings"
)

type RangeHint struct {
	Start int64 `json:"start"`
	End   int64 `json:"end"` // -1 means open-ended
	Valid bool  `json:"valid"`
}

// ParseRangeHint reads only a diagnostic hint. ServeContent remains the sole
// authority for HTTP Range semantics, including multipart and invalid ranges.
func ParseRangeHint(header string, size int64) RangeHint {
	if !strings.HasPrefix(header, "bytes=") || strings.Contains(header, ",") {
		return RangeHint{End: -1}
	}
	parts := strings.SplitN(strings.TrimPrefix(header, "bytes="), "-", 2)
	if len(parts) != 2 {
		return RangeHint{End: -1}
	}
	if parts[0] == "" {
		n, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil || n <= 0 || size <= 0 {
			return RangeHint{End: -1}
		}
		start := size - n
		if start < 0 {
			start = 0
		}
		return RangeHint{Start: start, End: size - 1, Valid: true}
	}
	start, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || start < 0 || start >= size {
		return RangeHint{End: -1}
	}
	if parts[1] == "" {
		return RangeHint{Start: start, End: -1, Valid: true}
	}
	end, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || end < start {
		return RangeHint{End: -1}
	}
	if end >= size {
		end = size - 1
	}
	return RangeHint{Start: start, End: end, Valid: true}
}

func Classify(method string, internal bool, hint RangeHint, size, previous int64, hadPrevious bool) string {
	if internal {
		return "PROBE_INTERNAL"
	}
	if method == http.MethodHead {
		return "HEAD_PROBE"
	}
	if !hint.Valid {
		return "UNKNOWN"
	}
	if hint.Start == 0 && hint.End >= 0 && hint.End < 2*MiB {
		return "HEAD_PROBE"
	}
	if size > 0 && hint.Start >= size-8*MiB {
		return "TAIL_INDEX"
	}
	if hadPrevious && previous > 0 && (hint.Start > previous+16*MiB || hint.Start+16*MiB < previous) {
		return "SEEK"
	}
	if hint.Start == 0 && hadPrevious {
		return "HEADER_REREAD"
	}
	return "PLAYBACK"
}
