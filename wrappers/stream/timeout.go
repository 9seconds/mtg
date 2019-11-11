package stream

import (
	"net"
	"time"

	"go.uber.org/zap"

	"mtg/conntypes"
)

const (
	timeoutRead  = 2 * time.Minute
	timeoutWrite = 2 * time.Minute
)

type wrapperTimeout struct {
	parent conntypes.StreamReadWriteCloser
}

func (w *wrapperTimeout) WriteTimeout(p []byte, timeout time.Duration) (int, error) {
	return w.parent.WriteTimeout(p, timeout)
}

func (w *wrapperTimeout) Write(p []byte) (int, error) {
	return w.parent.WriteTimeout(p, timeoutWrite)
}

func (w *wrapperTimeout) ReadTimeout(p []byte, timeout time.Duration) (int, error) {
	return w.parent.ReadTimeout(p, timeout)
}

func (w *wrapperTimeout) Read(p []byte) (int, error) {
	return w.parent.ReadTimeout(p, timeoutRead)
}

func (w *wrapperTimeout) Close() error {
	return w.parent.Close()
}

func (w *wrapperTimeout) Conn() net.Conn {
	return w.parent.Conn()
}

func (w *wrapperTimeout) Logger() *zap.SugaredLogger {
	return w.parent.Logger().Named("timeout")
}

func (w *wrapperTimeout) LocalAddr() *net.TCPAddr {
	return w.parent.LocalAddr()
}

func (w *wrapperTimeout) RemoteAddr() *net.TCPAddr {
	return w.parent.RemoteAddr()
}

func NewTimeout(parent conntypes.StreamReadWriteCloser) conntypes.StreamReadWriteCloser {
	return &wrapperTimeout{
		parent: parent,
	}
}
