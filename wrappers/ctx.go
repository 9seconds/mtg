package wrappers

import (
	"context"
	"net"

	"github.com/juju/errors"
)

type WrapCtx struct {
	cancel context.CancelFunc
	conn   WrapStreamReadWriteCloser
	ctx    context.Context
}

func (w *WrapCtx) Read(p []byte) (int, error) {
	select {
	case <-w.ctx.Done():
		return 0, errors.Annotate(w.ctx.Err(), "Read is failed because of closed context")
	default:
		n, err := w.conn.Read(p)
		if err != nil {
			w.cancel()
		}
		return n, err
	}
}

func (w *WrapCtx) Write(p []byte) (int, error) {
	select {
	case <-w.ctx.Done():
		return 0, errors.Annotate(w.ctx.Err(), "Write is failed because of closed context")
	default:
		n, err := w.conn.Write(p)
		if err != nil {
			w.cancel()
		}
		return n, err
	}
}

func (w *WrapCtx) LogDebug(msg string, data ...interface{}) {
	w.conn.LogDebug(msg, data...)
}

func (w *WrapCtx) LogInfo(msg string, data ...interface{}) {
	w.conn.LogInfo(msg, data...)
}

func (w *WrapCtx) LogWarn(msg string, data ...interface{}) {
	w.conn.LogWarn(msg, data...)
}

func (w *WrapCtx) LogError(msg string, data ...interface{}) {
	w.conn.LogError(msg, data...)
}

func (w *WrapCtx) LocalAddr() *net.TCPAddr {
	return w.conn.LocalAddr()
}

func (w *WrapCtx) RemoteAddr() *net.TCPAddr {
	return w.conn.RemoteAddr()
}

func (w *WrapCtx) Close() error {
	return w.conn.Close()
}

func NewCtx(ctx context.Context, cancel context.CancelFunc, conn WrapStreamReadWriteCloser) WrapStreamReadWriteCloser {
	return &WrapCtx{
		ctx:    ctx,
		cancel: cancel,
		conn:   conn,
	}
}
