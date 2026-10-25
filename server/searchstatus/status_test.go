package searchstatus

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCategorizationAndCancellation(t *testing.T) {
	for _, tc := range []struct {
		err  error
		want string
	}{{nil, "empty"}, {HTTP(401), "auth_required"}, {HTTP(503), "unavailable"}, {Invalid(), "invalid_response"}, {context.DeadlineExceeded, "timeout"}, {context.Canceled, "cancelled"}, {errors.New("private fixture"), "unavailable"}} {
		got := Outcome("fixture", 0, 0, time.Now(), tc.err)
		if got.State != tc.want {
			t.Fatal(got)
		}
	}
	var release []func()
	for i := 0; i < 4; i++ {
		done, err := Acquire(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		release = append(release, done)
	}
	defer func() {
		for _, done := range release {
			done()
		}
	}()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Acquire(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
