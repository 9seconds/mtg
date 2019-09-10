package wrappers

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"fmt"
	"net"
	"time"

	"go.uber.org/zap"
)

const blockCipherReadCurrentDataBufferSize = 1024 + 1 // +1 because telegram operates with blocks mod 4

type wrapperBlockCipher struct {
	buf bytes.Buffer

	parent    StreamReadWriteCloser
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
	return w.read(p, readAll)

}

func (w *wrapperBlockCipher) ReadTimeout(p []byte, timeout time.Duration) (int, error) {
	return w.read(p, readAllTimeout(timeout))
}

func (w *wrapperBlockCipher) read(p []byte, reader func(StreamReadWriteCloser) ([]byte, error)) (int, error) {
	if w.buf.Len() > 0 {
		return w.flush(p)
	}

	var buf []byte
	for len(buf) == 0 || len(buf)%aes.BlockSize != 0 {
		rv, err := reader(w.parent)
		if err != nil {
			return 0, fmt.Errorf("cannot read from socket: %w", err)
		}
		buf = append(buf, rv...)
	}

	w.decryptor.CryptBlocks(buf, buf)
	w.buf.Write(buf)

	return w.flush(p)
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

func readAll(src StreamReadWriteCloser) (rv []byte, err error) {
	buf := make([]byte, blockCipherReadCurrentDataBufferSize)
	n := blockCipherReadCurrentDataBufferSize

	for n == len(buf) {
		n, err = src.Read(buf)
		if err != nil {
			return nil, err
		}
		rv = append(rv, buf[:n]...)
	}

	return rv, nil
}

func readAllTimeout(timeout time.Duration) func(StreamReadWriteCloser) ([]byte, error) {
	return func(src StreamReadWriteCloser) (rv []byte, err error) {
		tmo := timeout
		buf := make([]byte, blockCipherReadCurrentDataBufferSize)
		n := blockCipherReadCurrentDataBufferSize

		for n == len(buf) {
			if tmo <= 0 {
				return nil, errors.New("timeout")
			}
			startTime := time.Now()
			n, err = src.ReadTimeout(buf, tmo)
			if err != nil {
				return nil, err
			}
			rv = append(rv, buf[:n]...)
			tmo -= time.Since(startTime)
		}

		return rv, nil
	}
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

func newBlockCipher(parent StreamReadWriteCloser, encryptor, decryptor cipher.BlockMode) StreamReadWriteCloser {
	return &wrapperBlockCipher{
		parent:    parent,
		encryptor: encryptor,
		decryptor: decryptor,
	}
}
