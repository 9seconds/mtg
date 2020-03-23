package stream

import (
	"bytes"
	"crypto/cipher"
	"fmt"
	"net"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/9seconds/mtg/conntypes"
)

var (
	poolWrapperObfuscated2WritePool = sync.Pool{
		New: func() interface{} {
			return &bytes.Buffer{}
		},
	}
)

func poolWrapperObfuscated2WritePoolAcquire() *bytes.Buffer {
	return poolWrapperObfuscated2WritePool.Get().(*bytes.Buffer)
}

func poolWrapperObfuscated2WritePoolRelease(buf *bytes.Buffer) {
	buf.Reset()
	poolWrapperObfuscated2WritePool.Put(buf)
}

type wrapperObfuscated2 struct {
	encryptor cipher.Stream
	decryptor cipher.Stream
	parent    conntypes.StreamReadWriteCloser
}

func (w *wrapperObfuscated2) ReadTimeout(p []byte, timeout time.Duration) (int, error) {
	n, err := w.parent.ReadTimeout(p, timeout)
	if err != nil {
		return 0, fmt.Errorf("cannot read stream ciphered data: %w", err)
	}

	w.decryptor.XORKeyStream(p, p[:n])

	return n, nil
}

func (w *wrapperObfuscated2) Read(p []byte) (int, error) {
	n, err := w.parent.Read(p)
	if err != nil {
		return n, err
	}

	w.decryptor.XORKeyStream(p, p[:n])

	return n, nil
}

func (w *wrapperObfuscated2) WriteTimeout(p []byte, timeout time.Duration) (int, error) {
	buffer := poolWrapperObfuscated2WritePoolAcquire()
	defer poolWrapperObfuscated2WritePoolRelease(buffer)

	buffer.Write(p)

	buf := buffer.Bytes()

	w.encryptor.XORKeyStream(buf, buf)

	return w.parent.WriteTimeout(buf, timeout)
}

func (w *wrapperObfuscated2) Write(p []byte) (int, error) {
	buffer := poolWrapperObfuscated2WritePoolAcquire()
	defer poolWrapperObfuscated2WritePoolRelease(buffer)

	buffer.Write(p)

	buf := buffer.Bytes()

	w.encryptor.XORKeyStream(buf, buf)

	return w.parent.Write(buf)
}

func (w *wrapperObfuscated2) Conn() net.Conn {
	return w.parent.Conn()
}

func (w *wrapperObfuscated2) Logger() *zap.SugaredLogger {
	return w.parent.Logger().Named("obfuscated2")
}

func (w *wrapperObfuscated2) LocalAddr() *net.TCPAddr {
	return w.parent.LocalAddr()
}

func (w *wrapperObfuscated2) RemoteAddr() *net.TCPAddr {
	return w.parent.RemoteAddr()
}

func (w *wrapperObfuscated2) Close() error {
	return w.parent.Close()
}

func NewObfuscated2(socket conntypes.StreamReadWriteCloser,
	encryptor, decryptor cipher.Stream) conntypes.StreamReadWriteCloser {
	return &wrapperObfuscated2{
		parent:    socket,
		encryptor: encryptor,
		decryptor: decryptor,
	}
}
