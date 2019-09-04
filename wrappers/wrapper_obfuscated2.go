package wrappers

import (
	"crypto/cipher"
	"net"
	"time"

	"github.com/juju/errors"
	"go.uber.org/zap"
)

type wrapperObfuscated2 struct {
	encryptor cipher.Stream
	decryptor cipher.Stream
	parent    StreamReadWriteCloser
}

func (w *wrapperObfuscated2) ReadTimeout(p []byte, timeout time.Duration) (int, error) {
	n, err := w.parent.ReadTimeout(p, timeout)
	if err != nil {
		return 0, errors.Annotate(err, "Cannot read stream ciphered data")
	}
	w.decryptor.XORKeyStream(p, p[:n])

	return n, nil
}

func (w *wrapperObfuscated2) Read(p []byte) (int, error) {
	n, err := w.parent.Read(p)
	if err != nil {
		return 0, errors.Annotate(err, "Cannot read stream ciphered data")
	}
	w.decryptor.XORKeyStream(p, p[:n])

	return n, nil
}

func (w *wrapperObfuscated2) WriteTimeout(p []byte, timeout time.Duration) (int, error) {
	buf := make([]byte, len(p))
	copy(buf, p)
	w.encryptor.XORKeyStream(buf, buf)

	return w.parent.WriteTimeout(buf, timeout)
}

func (w *wrapperObfuscated2) Write(p []byte) (int, error) {
	buf := make([]byte, len(p))
	copy(buf, p)
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

func NewObfuscated2(socket StreamReadWriteCloser, encryptor, decryptor cipher.Stream) StreamReadWriteCloser {
	return &wrapperObfuscated2{
		parent:    socket,
		encryptor: encryptor,
		decryptor: decryptor,
	}
}
