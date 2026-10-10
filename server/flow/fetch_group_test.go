package flow

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestSharedFetchCancellationPreservesOtherOwner(t *testing.T) {
	var group FetchGroup[int]
	var calls atomic.Int32
	started, finish := make(chan struct{}), make(chan struct{})
	fetch := func(ctx context.Context) (int, error) {
		calls.Add(1)
		close(started)
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-finish:
			return 42, nil
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	first := make(chan error, 1)
	go func() { _, err := group.Do(ctx, "same-policy", 1, fetch); first <- err }()
	<-started
	second := make(chan int, 1)
	go func() {
		value, err := group.Do(context.Background(), "same-policy", 1, fetch)
		if err != nil {
			value = -1
		}
		second <- value
	}()
	deadline := time.Now().Add(time.Second)
	for {
		group.mu.Lock()
		owners := group.entries["same-policy"].owners
		group.mu.Unlock()
		if owners == 2 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("second owner did not join")
		}
		time.Sleep(time.Millisecond)
	}
	cancel()
	if err := <-first; !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	close(finish)
	if value := <-second; value != 42 || calls.Load() != 1 {
		t.Fatal(value, calls.Load())
	}
}

func TestSharedFetchLastOwnerCancelsAndAdmissionIsBounded(t *testing.T) {
	var group FetchGroup[int]
	started, stopped := make(chan struct{}), make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		_, err := group.Do(ctx, "one", 1, func(shared context.Context) (int, error) {
			close(started)
			<-shared.Done()
			close(stopped)
			return 0, shared.Err()
		})
		result <- err
	}()
	<-started
	if _, err := group.Do(context.Background(), "different-credential", 1, func(context.Context) (int, error) { t.Error("admitted excess work"); return 0, nil }); !errors.Is(err, ErrFetchBusy) {
		t.Fatal(err)
	}
	cancel()
	if err := <-result; !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("orphan fetch was not cancelled")
	}
}
