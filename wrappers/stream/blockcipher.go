package stream

import (
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"net"
	"time"

	"github.com/9seconds/mtg/conntypes"
	"github.com/9seconds/mtg/utils"
	"go.uber.org/zap"
)

type wrapperBlockCipher struct {
	bufferedReader

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
	cipher := &wrapperBlockCipher{
		parent:    parent,
		encryptor: encryptor,
		decryptor: decryptor,
	}

	cipher.readFunc = func() ([]byte, error) {
		var currentBuffer []byte
		for len(currentBuffer) == 0 || len(currentBuffer)%aes.BlockSize != 0 {
			rv, err := utils.ReadFull(cipher.parent)
			if err != nil {
				return nil, fmt.Errorf("cannot read data: %w", err)
			}

			currentBuffer = append(currentBuffer, rv...)
		}
		cipher.decryptor.CryptBlocks(currentBuffer, currentBuffer)

		return currentBuffer, nil
	}

	return cipher
}
