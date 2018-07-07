package wrappers

import (
	"context"
	"net"

	"github.com/juju/errors"
)

type Ctx struct {
	cancel context.CancelFunc
	conn   WrapStreamReadWriteCloser
	ctx    context.Context
}

func (c *Ctx) Read(p []byte) (int, error) {
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

func (c *Ctx) Write(p []byte) (int, error) {
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

func (c *Ctx) LogDebug(msg string, data ...interface{}) {
	c.conn.LogDebug(msg, data...)
}

func (c *Ctx) LogInfo(msg string, data ...interface{}) {
	c.conn.LogInfo(msg, data...)
}

func (c *Ctx) LogWarn(msg string, data ...interface{}) {
	c.conn.LogWarn(msg, data...)
}

func (c *Ctx) LogError(msg string, data ...interface{}) {
	c.conn.LogError(msg, data...)
}

func (c *Ctx) LocalAddr() *net.TCPAddr {
	return c.conn.LocalAddr()
}

func (c *Ctx) RemoteAddr() *net.TCPAddr {
	return c.conn.RemoteAddr()
}

func (c *Ctx) Close() error {
	return c.conn.Close()
}

func NewCtx(ctx context.Context, cancel context.CancelFunc, conn WrapStreamReadWriteCloser) WrapStreamReadWriteCloser {
	return &Ctx{
		ctx:    ctx,
		cancel: cancel,
		conn:   conn,
	}
}
