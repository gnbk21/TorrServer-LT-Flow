//go:build windows

package netchange

import "testing"

func TestWatchCancellationDoesNotRetainSubscribers(t *testing.T) {
	for i := 0; i < 20; i++ {
		_, close := Watch()
		close()
		close()
	}
	n := 0
	subscribers.Range(func(_, _ any) bool { n++; return true })
	if n != 0 {
		t.Fatalf("retained %d subscriptions", n)
	}
}
