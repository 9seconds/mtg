package proxy

import (
	"context"
	"io"

	"github.com/juju/errors"
)

// CtxReadWriteCloser wraps underlying connection and does management of the
// context and its cancel function.
type CtxReadWriteCloser struct {
	ctx    context.Context
	conn   io.ReadWriteCloser
	cancel context.CancelFunc
}

// Read reads from connection
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

// Write writes into connection.
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

// Close closes underlying connection.
func (c *CtxReadWriteCloser) Close() error {
	return c.conn.Close()
}

func newCtxReadWriteCloser(ctx context.Context, cancel context.CancelFunc, conn io.ReadWriteCloser) io.ReadWriteCloser {
	return &CtxReadWriteCloser{
		conn:   conn,
		ctx:    ctx,
		cancel: cancel,
	}
}
