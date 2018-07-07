package wrappers

import (
	"crypto/cipher"
	"net"

	"github.com/juju/errors"
)

type WrapStreamCipher struct {
	encryptor cipher.Stream
	decryptor cipher.Stream
	conn      WrapStreamReadWriteCloser
}

func (w *WrapStreamCipher) Read(p []byte) (int, error) {
	n, err := w.conn.Read(p)
	if err != nil {
		return 0, errors.Annotate(err, "Cannot read stream ciphered data")
	}
	w.decryptor.XORKeyStream(p, p[:n])

	return n, nil
}

func (w *WrapStreamCipher) Write(p []byte) (int, error) {
	encrypted := make([]byte, len(p))
	w.encryptor.XORKeyStream(encrypted, p)

	return w.conn.Write(encrypted)
}

func (w *WrapStreamCipher) LogDebug(msg string, data ...interface{}) {
	w.conn.LogDebug(msg, data...)
}

func (w *WrapStreamCipher) LogInfo(msg string, data ...interface{}) {
	w.conn.LogInfo(msg, data...)
}

func (w *WrapStreamCipher) LogWarn(msg string, data ...interface{}) {
	w.conn.LogWarn(msg, data...)
}

func (w *WrapStreamCipher) LogError(msg string, data ...interface{}) {
	w.conn.LogError(msg, data...)
}

func (w *WrapStreamCipher) LocalAddr() *net.TCPAddr {
	return w.conn.LocalAddr()
}

func (w *WrapStreamCipher) RemoteAddr() *net.TCPAddr {
	return w.conn.RemoteAddr()
}

func (w *WrapStreamCipher) Close() error {
	return w.conn.Close()
}

func NewStreamCipher(conn WrapStreamReadWriteCloser, encryptor, decryptor cipher.Stream) WrapStreamReadWriteCloser {
	return &WrapStreamCipher{
		conn:      conn,
		encryptor: encryptor,
		decryptor: decryptor,
	}
}
