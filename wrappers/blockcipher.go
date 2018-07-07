package wrappers

import (
	"crypto/aes"
	"crypto/cipher"
	"net"

	"github.com/9seconds/mtg/utils"
	"github.com/juju/errors"
)

type WrapBlockCipher struct {
	BufferedReader

	conn      WrapStreamReadWriteCloser
	encryptor cipher.BlockMode
	decryptor cipher.BlockMode
}

func (w *WrapBlockCipher) Read(p []byte) (int, error) {
	return w.BufferedRead(p, func() error {
		var buf []byte

		for len(buf) == 0 || len(buf)%aes.BlockSize != 0 {
			rv, err := utils.ReadCurrentData(w.conn)
			if err != nil {
				return errors.Annotate(err, "Cannot read from socket")
			}
			buf = append(buf, rv...)
		}

		w.decryptor.CryptBlocks(buf, buf)
		w.Buffer.Write(buf)

		return nil
	})
}

func (w *WrapBlockCipher) Write(p []byte) (int, error) {
	if len(p)%aes.BlockSize > 0 {
		return 0, errors.Errorf("Incorrect block size %d", len(p))
	}

	encrypted := make([]byte, len(p))
	w.encryptor.CryptBlocks(encrypted, p)

	return w.conn.Write(encrypted)
}

func (w *WrapBlockCipher) LogDebug(msg string, data ...interface{}) {
	w.conn.LogDebug(msg, data...)
}

func (w *WrapBlockCipher) LogInfo(msg string, data ...interface{}) {
	w.conn.LogInfo(msg, data...)
}

func (w *WrapBlockCipher) LogWarn(msg string, data ...interface{}) {
	w.conn.LogWarn(msg, data...)
}

func (w *WrapBlockCipher) LogError(msg string, data ...interface{}) {
	w.conn.LogError(msg, data...)
}

func (w *WrapBlockCipher) LocalAddr() *net.TCPAddr {
	return w.conn.LocalAddr()
}

func (w *WrapBlockCipher) RemoteAddr() *net.TCPAddr {
	return w.conn.RemoteAddr()
}

func (w *WrapBlockCipher) Close() error {
	return w.conn.Close()
}

func NewWrapBlockCipher(conn WrapStreamReadWriteCloser, encryptor, decryptor cipher.BlockMode) WrapStreamReadWriteCloser {
	return &WrapBlockCipher{
		BufferedReader: NewBufferedReader(),
		conn:           conn,
		encryptor:      encryptor,
		decryptor:      decryptor,
	}
}
