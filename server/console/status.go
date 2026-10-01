package console

import (
	"context"
	"time"
)

// RunStatus emits immediately, on state changes (sampled every five seconds),
// and at the requested heartbeat interval. Metrics changing alone do not flood
// the log. The owner must cancel and join it before closing its output.
func RunStatus(ctx context.Context, interval time.Duration, state func() string, message func() string, emit func(string)) {
	if interval <= 0 {
		return
	}
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	var lastKey string
	var lastAt time.Time
	for {
		if ctx.Err() != nil {
			return
		}
		key := state()
		now := time.Now()
		if ShouldReport(now, lastAt, interval, key, lastKey) {
			if ctx.Err() != nil {
				return
			}
			emit(message())
			lastKey, lastAt = key, now
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func ShouldReport(now, lastAt time.Time, interval time.Duration, key, lastKey string) bool {
	return interval > 0 && (lastAt.IsZero() || key != lastKey || now.Sub(lastAt) >= interval)
}
