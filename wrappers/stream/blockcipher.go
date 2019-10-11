package stream

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"net"
	"time"

	"go.uber.org/zap"

	"github.com/9seconds/mtg/conntypes"
	"github.com/9seconds/mtg/utils"
)

type wrapperBlockCipher struct {
	buf bytes.Buffer

	parent    conntypes.StreamReadWriteCloser
	encryptor cipher.BlockMode
	decryptor cipher.BlockMode
}

func (w *wrapperBlockCipher) Write(p []byte) (int, error) {
	encrypted, err := w.encrypt(p)
	if err != nil {
		return 0, err
	}

	return w.parent.Write(encrypted)
}

func (w *wrapperBlockCipher) WriteTimeout(p []byte, timeout time.Duration) (int, error) {
	encrypted, err := w.encrypt(p)
	if err != nil {
		return 0, err
	}

	return w.parent.WriteTimeout(encrypted, timeout)
}

func (w *wrapperBlockCipher) Read(p []byte) (int, error) {
	if w.buf.Len() > 0 {
		return w.flush(p)
	}

	var currentBuffer []byte
	for len(currentBuffer) == 0 || len(currentBuffer)%aes.BlockSize != 0 {
		rv, err := utils.ReadFull(w.parent)
		if err != nil {
			return 0, fmt.Errorf("cannot read data: %w", err)
		}

		currentBuffer = append(currentBuffer, rv...)
	}

	w.decryptor.CryptBlocks(currentBuffer, currentBuffer)
	w.buf.Write(currentBuffer)

	return w.flush(p)
}

func (w *wrapperBlockCipher) ReadTimeout(p []byte, timeout time.Duration) (int, error) {
	return w.Read(p)
}

func (w *wrapperBlockCipher) flush(p []byte) (int, error) {
	if w.buf.Len() > len(p) {
		return w.buf.Read(p)
	}

	sizeToReturn := w.buf.Len()
	copy(p, w.buf.Bytes())
	w.buf.Reset()

	return sizeToReturn, nil
}

func (w *wrapperBlockCipher) encrypt(p []byte) ([]byte, error) {
	if len(p)%aes.BlockSize > 0 {
		return nil, fmt.Errorf("incorrect block size %d", len(p))
	}

	encrypted := make([]byte, len(p))
	w.encryptor.CryptBlocks(encrypted, p)

	return encrypted, nil
}

func (w *wrapperBlockCipher) Close() error {
	return w.parent.Close()
}

func (w *wrapperBlockCipher) Conn() net.Conn {
	return w.parent.Conn()
}

func (w *wrapperBlockCipher) Logger() *zap.SugaredLogger {
	return w.parent.Logger().Named("block-cipher")
}

func (w *wrapperBlockCipher) LocalAddr() *net.TCPAddr {
	return w.parent.LocalAddr()
}

func (w *wrapperBlockCipher) RemoteAddr() *net.TCPAddr {
	return w.parent.RemoteAddr()
}

func newBlockCipher(parent conntypes.StreamReadWriteCloser,
	encryptor, decryptor cipher.BlockMode) conntypes.StreamReadWriteCloser {
	return &wrapperBlockCipher{
		parent:    parent,
		encryptor: encryptor,
		decryptor: decryptor,
	}
}
