package stream

import (
	"net"
	"time"

	"go.uber.org/zap"

	"mtg/conntypes"
	"mtg/stats"
)

type wrapperTrafficStats struct {
	parent conntypes.StreamReadWriteCloser
}

func (w *wrapperTrafficStats) Write(p []byte) (int, error) {
	n, err := w.parent.Write(p)
	stats.Stats.EgressTraffic(n)

	return n, err
}

func (w *wrapperTrafficStats) WriteTimeout(p []byte, timeout time.Duration) (int, error) {
	n, err := w.parent.WriteTimeout(p, timeout)
	stats.Stats.EgressTraffic(n)

	return n, err
}

func (w *wrapperTrafficStats) Read(p []byte) (int, error) {
	n, err := w.parent.Read(p)
	stats.Stats.IngressTraffic(n)

	return n, err
}

func (w *wrapperTrafficStats) ReadTimeout(p []byte, timeout time.Duration) (int, error) {
	n, err := w.parent.ReadTimeout(p, timeout)
	stats.Stats.IngressTraffic(n)

	return n, err
}

func (w *wrapperTrafficStats) Conn() net.Conn {
	return w.parent.Conn()
}

func (w *wrapperTrafficStats) Logger() *zap.SugaredLogger {
	return w.parent.Logger().Named("stats-traffic")
}

func (w *wrapperTrafficStats) LocalAddr() *net.TCPAddr {
	return w.parent.LocalAddr()
}

func (w *wrapperTrafficStats) RemoteAddr() *net.TCPAddr {
	return w.parent.RemoteAddr()
}

func (w *wrapperTrafficStats) Close() error {
	return w.parent.Close()
}

func NewTrafficStats(parent conntypes.StreamReadWriteCloser) conntypes.StreamReadWriteCloser {
	return &wrapperTrafficStats{parent}
}
