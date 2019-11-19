package stream

import (
	"context"
	"fmt"
	"net"
	"time"

	"go.uber.org/zap"

	"github.com/9seconds/mtg/conntypes"
)

type wrapperCtx struct {
	parent conntypes.StreamReadWriteCloser
	ctx    context.Context
	cancel context.CancelFunc
}

func (w *wrapperCtx) WriteTimeout(p []byte, timeout time.Duration) (int, error) {
	select {
	case <-w.ctx.Done():
		w.Close()
		return 0, fmt.Errorf("cannot write because context was closed: %w", w.ctx.Err())
	default:
		return w.parent.WriteTimeout(p, timeout)
	}
}

func (w *wrapperCtx) Write(p []byte) (int, error) {
	select {
	case <-w.ctx.Done():
		w.Close()
		return 0, fmt.Errorf("cannot write because context was closed: %w", w.ctx.Err())
	default:
		return w.parent.Write(p)
	}
}

func (w *wrapperCtx) ReadTimeout(p []byte, timeout time.Duration) (int, error) {
	select {
	case <-w.ctx.Done():
		w.Close()
		return 0, fmt.Errorf("cannot write because context was closed: %w", w.ctx.Err())
	default:
		return w.parent.ReadTimeout(p, timeout)
	}
}

func (w *wrapperCtx) Read(p []byte) (int, error) {
	select {
	case <-w.ctx.Done():
		w.Close()
		return 0, fmt.Errorf("cannot write because context was closed: %w", w.ctx.Err())
	default:
		return w.parent.Read(p)
	}
}

func (w *wrapperCtx) Close() error {
	w.cancel()
	return w.parent.Close()
}

func (w *wrapperCtx) Conn() net.Conn {
	return w.parent.Conn()
}

func (w *wrapperCtx) Logger() *zap.SugaredLogger {
	return w.parent.Logger().Named("ctx")
}

func (w *wrapperCtx) LocalAddr() *net.TCPAddr {
	return w.parent.LocalAddr()
}

func (w *wrapperCtx) RemoteAddr() *net.TCPAddr {
	return w.parent.RemoteAddr()
}

func NewCtx(ctx context.Context,
	cancel context.CancelFunc,
	parent conntypes.StreamReadWriteCloser) conntypes.StreamReadWriteCloser {
	return &wrapperCtx{
		parent: parent,
		ctx:    ctx,
		cancel: cancel,
	}
}
