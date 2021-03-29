package relay

import (
	"context"
	"io"
)

type conn struct {
	io.ReadWriteCloser

	ctx         context.Context
	tickChannel chan struct{}
}

func (c conn) Read(p []byte) (int, error) {
	n, err := c.ReadWriteCloser.Read(p)

	select {
	case <-c.ctx.Done():
	case c.tickChannel <- struct{}{}:
	}

	return n, err // nolint: wrapcheck
}

func (c conn) Write(p []byte) (int, error) {
	n, err := c.ReadWriteCloser.Write(p)

	select {
	case <-c.ctx.Done():
	case c.tickChannel <- struct{}{}:
	}

	return n, err // nolint: wrapcheck
}
