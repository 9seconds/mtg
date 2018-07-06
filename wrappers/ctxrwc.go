package wrappers

import (
	"context"
	"net"

	"github.com/juju/errors"
)

// CtxReadWriteCloser wraps underlying connection and does management of the
// context and its cancel function.
type CtxReadWriteCloserWithAddr struct {
	ctx    context.Context
	conn   ReadWriteCloserWithAddr
	cancel context.CancelFunc
}

// Read reads from connection
func (c *CtxReadWriteCloserWithAddr) Read(p []byte) (int, error) {
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
func (c *CtxReadWriteCloserWithAddr) Write(p []byte) (int, error) {
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
func (c *CtxReadWriteCloserWithAddr) Close() error {
	return c.conn.Close()
}

func (c *CtxReadWriteCloserWithAddr) LocalAddr() *net.TCPAddr {
	return c.conn.LocalAddr()
}

func (c *CtxReadWriteCloserWithAddr) RemoteAddr() *net.TCPAddr {
	return c.conn.RemoteAddr()
}

func (c *CtxReadWriteCloserWithAddr) SocketID() string {
	return c.conn.SocketID()
}

// NewCtxRWC returns ReadWriteCloser which respects given context,
// cancellation etc.
func NewCtxRWC(ctx context.Context, cancel context.CancelFunc, conn ReadWriteCloserWithAddr) ReadWriteCloserWithAddr {
	return &CtxReadWriteCloserWithAddr{
		conn:   conn,
		ctx:    ctx,
		cancel: cancel,
	}
}
