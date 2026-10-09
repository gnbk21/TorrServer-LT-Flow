package flow

import (
	"context"
	"time"
)

// RetryTransient bounds attempts and waits. Operations must return promptly;
// cancellation cannot interrupt a native call that does not accept a context.
func RetryTransient(ctx context.Context, operation func() error, transient func(error) bool) error {
	for attempt := 0; ; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		err := operation()
		if err == nil || attempt == 2 || !transient(err) {
			return err
		}
		timer := time.NewTimer(time.Duration(1<<attempt) * 200 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}
