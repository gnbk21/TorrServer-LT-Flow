package flow

import (
	"context"
	"errors"
	"testing"
)

func TestRetryTransientBoundsAndCancellation(t *testing.T) {
	temporary := errors.New("temporary")
	permanent := errors.New("permanent")
	for _, tc := range []struct {
		name           string
		failures, want int
		failure        error
	}{
		{"immediate", 0, 1, temporary}, {"recover", 1, 2, temporary},
		{"bounded", 9, 3, temporary}, {"permanent", 9, 1, permanent},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			err := RetryTransient(context.Background(), func() error {
				calls++
				if calls <= tc.failures {
					return tc.failure
				}
				return nil
			}, func(err error) bool { return errors.Is(err, temporary) })
			if calls != tc.want || ((err == nil) != (tc.failures < tc.want)) {
				t.Fatalf("calls %d, error %v", calls, err)
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	err := RetryTransient(ctx, func() error { calls++; cancel(); return temporary }, func(error) bool { return true })
	if calls != 1 || !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation: %d, %v", calls, err)
	}
}
