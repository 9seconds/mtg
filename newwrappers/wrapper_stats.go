package newwrappers

import (
	"net"
	"time"

	"go.uber.org/zap"

	"github.com/9seconds/mtg/newstats"
)

type wrapperStats struct {
	parent StreamReadWriteCloser
}

func (w *wrapperStats) Write(p []byte) (int, error) {
	n, err := w.parent.Write(p)
	newstats.S.EgressTraffic(n)

	return n, err
}

func (w *wrapperStats) WriteTimeout(p []byte, timeout time.Duration) (int, error) {
	n, err := w.parent.WriteTimeout(p, timeout)
	newstats.S.EgressTraffic(n)

	return n, err
}

func (w *wrapperStats) Read(p []byte) (int, error) {
	n, err := w.parent.Read(p)
	newstats.S.IngressTraffic(n)

	return n, err
}

func (w *wrapperStats) ReadTimeout(p []byte, timeout time.Duration) (int, error) {
	n, err := w.parent.ReadTimeout(p, timeout)
	newstats.S.IngressTraffic(n)

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

func NewTraffic(parent StreamReadWriteCloser) StreamReadWriteCloser {
	return &wrapperStats{parent}
}
