package server

import (
	"context"
	"io"

	"github.com/juju/errors"
)

type CtxReadWriteCloser struct {
	ctx    context.Context
	conn   io.ReadWriteCloser
	cancel context.CancelFunc
}

func (c *CtxReadWriteCloser) Read(p []byte) (int, error) {
	select {
	case <-c.ctx.Done():
		return 0, errors.Annotate(c.ctx.Err(), "Read is failed because of closed context")
	default:
		n, err := c.conn.Read(p)
		if err != nil {
			c.cancel()
		}
		return n, err
	}
}

func (c *CtxReadWriteCloser) Write(p []byte) (int, error) {
	select {
	case <-c.ctx.Done():
		return 0, errors.Annotate(c.ctx.Err(), "Write is failed because of closed context")
	default:
		n, err := c.conn.Write(p)
		if err != nil {
			c.cancel()
		}
		return n, err
	}
}

func (c *CtxReadWriteCloser) Close() error {
	return c.conn.Close()
}

func newCtxReadWriteCloser(conn io.ReadWriteCloser, ctx context.Context, cancel context.CancelFunc) io.ReadWriteCloser {
	return &CtxReadWriteCloser{
		conn:   conn,
		ctx:    ctx,
		cancel: cancel,
	}
}
