package torr

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"
)

func TestCancelledPlayDoesNotAttachReaderAfterPreload(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	request := httptest.NewRequest("GET", "/stream?play", nil).WithContext(ctx)
	response := httptest.NewRecorder()
	before := GetActiveStreams()
	var torrent *Torrent // There must be no torrent/cache access after cancellation.
	if err := torrent.Stream(1, request, response); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled playback entered the media path: %v", err)
	}
	if GetActiveStreams() != before || response.Body.Len() != 0 || len(response.Header()) != 0 {
		t.Fatal("cancelled playback created stream state or wrote a response")
	}
}
