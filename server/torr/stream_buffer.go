package torr

import (
	"bufio"
	"context"
	"io"
)

// Adapted from LT commit 32170776491adea2e2187726219544d6c85d8ead.
// This bounded transport buffer amortizes cache reads; the piece cache remains
// authoritative. Allocate lazily so HEAD and rejected ranges use no buffer.
const streamBufferSize = 1 << 20

type streamConsumption interface {
	TrackBufferedConsumption()
	ConsumeBuffered(int)
}

type bufferedStreamReader struct {
	source      io.ReadSeeker
	buffer      *bufio.Reader
	size        int
	ctx         context.Context
	consumption streamConsumption
}

func newBufferedStreamReader(source io.ReadSeeker, size int) *bufferedStreamReader {
	r := &bufferedStreamReader{source: source, size: size}
	if c, ok := source.(streamConsumption); ok {
		r.consumption = c
		c.TrackBufferedConsumption()
	}
	return r
}

func (r *bufferedStreamReader) Read(p []byte) (int, error) {
	if r.ctx != nil && r.ctx.Err() != nil {
		return 0, r.ctx.Err()
	}
	if len(p) == 0 {
		return 0, nil
	}
	if r.buffer == nil {
		r.buffer = bufio.NewReaderSize(r.source, r.size)
	}
	n, err := r.buffer.Read(p)
	if n > 0 && r.consumption != nil {
		r.consumption.ConsumeBuffered(n)
	}
	return n, err
}

func (r *bufferedStreamReader) Seek(offset int64, whence int) (int64, error) {
	if whence == io.SeekCurrent && r.buffer != nil {
		offset -= int64(r.buffer.Buffered())
	}
	pos, err := r.source.Seek(offset, whence)
	if err == nil && r.buffer != nil {
		r.buffer.Reset(r.source)
	}
	return pos, err
}
