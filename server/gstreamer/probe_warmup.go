//go:build gst

package gstreamer

import (
	"context"
	"errors"
	"time"
)

// Adapted from LT fbff205942d75d8266350740f295e732da76daf0. A cold
// container may need its header and seek index before discovery can succeed.
const (
	probeWarmupHeadBytes = 512 << 10
	probeWarmupTailBytes = 256 << 10
	probeWarmupTimeout   = 45 * time.Second
)

func probeRetryableError(err error) bool {
	return errors.Is(err, ErrProbeUnavailable) || errors.Is(err, context.DeadlineExceeded)
}

func warmProbeSource(parent context.Context, sourceURL string, fileSize int64) bool {
	ctx, cancel := context.WithTimeout(parent, probeWarmupTimeout)
	defer cancel()
	head := probeWarmupHeadBytes
	if fileSize > 0 && fileSize < int64(head) {
		head = int(fileSize)
	}
	if _, err := readHTTPRange(ctx, sourceURL, 0, head); err != nil {
		return false
	}
	if fileSize > probeWarmupHeadBytes+probeWarmupTailBytes {
		// The head alone may suffice. A tail failure remains best effort.
		_, _ = readHTTPRange(ctx, sourceURL, fileSize-probeWarmupTailBytes, probeWarmupTailBytes)
	}
	return ctx.Err() == nil
}
