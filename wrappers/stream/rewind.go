package stream

import (
	"bytes"
	"io"
	"net"
	"sync"
	"time"

	"github.com/9seconds/mtg/conntypes"
	"go.uber.org/zap"
)

type ReadWriteCloseRewinder interface {
	conntypes.StreamReadWriteCloser
	Rewind()
}

type wrapperRewind struct {
	parent       conntypes.StreamReadWriteCloser
	activeReader io.Reader
	buf          bytes.Buffer
	mutex        sync.Mutex
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

	return w.activeReader.Read(p)
}

func (w *wrapperRewind) ReadTimeout(p []byte, _ time.Duration) (int, error) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	return w.activeReader.Read(p)
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
	w.activeReader = io.MultiReader(&w.buf, w.parent)
	w.mutex.Unlock()
}

func NewRewind(parent conntypes.StreamReadWriteCloser) ReadWriteCloseRewinder {
	rv := &wrapperRewind{
		parent: parent,
	}
	rv.activeReader = io.TeeReader(parent, &rv.buf)

	return rv
}
