// Package flow contains playback estimates. It does not own a second cache.
package flow

import (
	"math"
	"strconv"
	"strings"
)

const MiB int64 = 1 << 20

type Estimate struct {
	BytesPerSecond float64 `json:"bytes_per_second"`
	Source         string  `json:"source"`
	Confidence     string  `json:"confidence"`
}

func Provisional() Estimate {
	return Estimate{Source: "PROVISIONAL", Confidence: "low"}
}

// MediaEstimate prefers credible format bitrate, then file size divided by
// duration. A zero rate remains provisional; it is never guessed from a name.
func MediaEstimate(fileBytes int64, durationSeconds float64, formatBitrate string) Estimate {
	if bits, err := strconv.ParseFloat(strings.TrimSpace(formatBitrate), 64); err == nil && bits >= 1000 && !math.IsInf(bits, 0) && !math.IsNaN(bits) {
		return Estimate{BytesPerSecond: bits / 8, Source: "PROBED", Confidence: "medium"}
	}
	if fileBytes > 0 && durationSeconds > 0 && !math.IsInf(durationSeconds, 0) && !math.IsNaN(durationSeconds) {
		return Estimate{BytesPerSecond: float64(fileBytes) / durationSeconds, Source: "DERIVED", Confidence: "medium"}
	}
	return Provisional()
}

func StartupTarget(e Estimate, seconds, minMB, maxMB, safetyPct int) int64 {
	minBytes, maxBytes := int64(minMB)*MiB, int64(maxMB)*MiB
	if minBytes < MiB {
		minBytes = MiB
	}
	if maxBytes < minBytes {
		maxBytes = minBytes
	}
	if e.BytesPerSecond <= 0 {
		return minBytes
	}
	target := e.BytesPerSecond * float64(seconds) * float64(safetyPct) / 100
	if target < float64(minBytes) {
		return minBytes
	}
	if target > float64(maxBytes) {
		return maxBytes
	}
	return int64(math.Ceil(target))
}

func BufferSeconds(contiguousBytes int64, e Estimate) float64 {
	if contiguousBytes <= 0 || e.BytesPerSecond <= 0 {
		return 0
	}
	return float64(contiguousBytes) / e.BytesPerSecond
}

func Sustainability(downloadBytesPerSecond float64, e Estimate) float64 {
	if downloadBytesPerSecond <= 0 || e.BytesPerSecond <= 0 {
		return 0
	}
	return downloadBytesPerSecond / e.BytesPerSecond
}
