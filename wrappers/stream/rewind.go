package stream

import (
	"bytes"
	"io"
	"net"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/9seconds/mtg/conntypes"
)

type ReadWriteCloseRewinder interface {
	conntypes.StreamReadWriteCloser
	Rewind()
}

type wrapperRewind struct {
	parent   conntypes.StreamReadWriteCloser
	buf      bytes.Buffer
	mutex    sync.Mutex
	rewinded bool
}

func (w *wrapperRewind) Write(p []byte) (int, error) {
	return w.parent.Write(p)
}

func (w *wrapperRewind) WriteTimeout(p []byte, timeout time.Duration) (int, error) {
	return w.parent.WriteTimeout(p, timeout)
}

func (w *wrapperRewind) Read(p []byte) (int, error) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if w.rewinded {
		if n, err := w.buf.Read(p); err != io.EOF {
			return n, err
		}
	}

	n, err := w.parent.Read(p)
	w.buf.Write(p[:n])

	return n, err
}

func (w *wrapperRewind) ReadTimeout(p []byte, timeout time.Duration) (int, error) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if w.rewinded {
		if n, err := w.buf.Read(p); err != io.EOF {
			return n, err
		}
	}

	n, err := w.parent.ReadTimeout(p, timeout)
	w.buf.Write(p[:n])

	return n, err
}

func (w *wrapperRewind) Conn() net.Conn {
	return w.parent.Conn()
}

func (w *wrapperRewind) Logger() *zap.SugaredLogger {
	return w.parent.Logger().Named("rewinded")
}

func (w *wrapperRewind) LocalAddr() *net.TCPAddr {
	return w.parent.LocalAddr()
}

func (w *wrapperRewind) RemoteAddr() *net.TCPAddr {
	return w.parent.RemoteAddr()
}

func (w *wrapperRewind) Close() error {
	w.buf.Reset()
	return w.parent.Close()
}

func (w *wrapperRewind) Rewind() {
	w.mutex.Lock()
	w.rewinded = true
	w.mutex.Unlock()
}

func NewRewind(parent conntypes.StreamReadWriteCloser) ReadWriteCloseRewinder {
	return &wrapperRewind{
		parent: parent,
	}
}
