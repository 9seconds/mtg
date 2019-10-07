package wrappers

import (
	"net"
	"time"

	"go.uber.org/zap"

	"github.com/9seconds/mtg/conntypes"
	"github.com/9seconds/mtg/stats"
)

type wrapperStats struct {
	parent conntypes.StreamReadWriteCloser
}

func (w *wrapperStats) Write(p []byte) (int, error) {
	n, err := w.parent.Write(p)
	stats.S.EgressTraffic(n)

	return n, err
}

func (w *wrapperStats) WriteTimeout(p []byte, timeout time.Duration) (int, error) {
	n, err := w.parent.WriteTimeout(p, timeout)
	stats.S.EgressTraffic(n)

	return n, err
}

func (w *wrapperStats) Read(p []byte) (int, error) {
	n, err := w.parent.Read(p)
	stats.S.IngressTraffic(n)

	return n, err
}

func (w *wrapperStats) ReadTimeout(p []byte, timeout time.Duration) (int, error) {
	n, err := w.parent.ReadTimeout(p, timeout)
	stats.S.IngressTraffic(n)

	return n, err
}

func (w *wrapperStats) Conn() net.Conn {
	return w.parent.Conn()
}

func (w *wrapperStats) Logger() *zap.SugaredLogger {
	return w.parent.Logger().Named("traffic")
}

func (w *wrapperStats) LocalAddr() *net.TCPAddr {
	return w.parent.LocalAddr()
}

func (w *wrapperStats) RemoteAddr() *net.TCPAddr {
	return w.parent.RemoteAddr()
}

func (w *wrapperStats) Close() error {
	return w.parent.Close()
}

func NewTraffic(parent conntypes.StreamReadWriteCloser) conntypes.StreamReadWriteCloser {
	return &wrapperStats{parent}
}
