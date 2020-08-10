package stream

import (
	"net"
	"sync"
	"time"

	"github.com/9seconds/mtg/conntypes"
	"github.com/9seconds/mtg/stats"
	"go.uber.org/zap"
)

type wrapperTelegramStats struct {
	parent conntypes.StreamReadWriteCloser
	dc     conntypes.DC
	once   sync.Once
}

func (w *wrapperTelegramStats) Write(p []byte) (int, error) {
	return w.parent.Write(p)
}

func (w *wrapperTelegramStats) WriteTimeout(p []byte, timeout time.Duration) (int, error) {
	return w.parent.WriteTimeout(p, timeout)
}

func (w *wrapperTelegramStats) Read(p []byte) (int, error) {
	return w.parent.Read(p)
}

func (w *wrapperTelegramStats) ReadTimeout(p []byte, timeout time.Duration) (int, error) {
	return w.parent.ReadTimeout(p, timeout)
}

func (w *wrapperTelegramStats) Conn() net.Conn {
	return w.parent.Conn()
}

func (w *wrapperTelegramStats) Logger() *zap.SugaredLogger {
	return w.parent.Logger().Named("stats-telegram")
}

func (w *wrapperTelegramStats) LocalAddr() *net.TCPAddr {
	return w.parent.LocalAddr()
}

func (w *wrapperTelegramStats) RemoteAddr() *net.TCPAddr {
	return w.parent.RemoteAddr()
}

func (w *wrapperTelegramStats) Close() error {
	var err error

	w.once.Do(func() {
		err = w.parent.Close()
		stats.Stats.TelegramDisconnected(w.dc, w.RemoteAddr())
	})

	return err
}

func NewTelegramStats(dc conntypes.DC, parent conntypes.StreamReadWriteCloser) conntypes.StreamReadWriteCloser {
	conn := &wrapperTelegramStats{
		parent: parent,
		dc:     dc,
	}

	stats.Stats.TelegramConnected(dc, parent.RemoteAddr())

	return conn
}
