// Package searchstatus bounds configured-indexer work and exposes safe outcomes.
package searchstatus

import (
	"context"
	"errors"
	"net"
	"time"
)

var slots = make(chan struct{}, 4)

func Acquire(ctx context.Context) (func(), error) {
	select {
	case slots <- struct{}{}:
		return func() { <-slots }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

type Error struct{ Code string }

func (e Error) Error() string { return e.Code }
func Invalid() error          { return Error{"invalid_response"} }
func HTTP(code int) error {
	if code == 401 || code == 403 {
		return Error{"auth_required"}
	}
	return Error{"unavailable"}
}

type Status struct {
	Source    string `json:"source"`
	Index     int    `json:"index"`
	State     string `json:"state"`
	ElapsedMS int64  `json:"elapsed_ms"`
}

func Outcome(source string, index, count int, started time.Time, err error) Status {
	state := "ok"
	if count == 0 {
		state = "empty"
	}
	if err != nil {
		state = "unavailable"
		var coded Error
		var network net.Error
		switch {
		case errors.As(err, &coded):
			state = coded.Code
		case errors.Is(err, context.DeadlineExceeded):
			state = "timeout"
		case errors.Is(err, context.Canceled):
			state = "cancelled"
		case errors.As(err, &network) && network.Timeout():
			state = "timeout"
		}
	}
	return Status{source, index, state, time.Since(started).Milliseconds()}
}
