package server

import (
	"bytes"
	"io"
)

type Cipher interface {
	Encrypt([]byte) []byte
	Decrypt([]byte) []byte
}

type CipherReadWriteCloser struct {
	crypt Cipher
	conn  io.ReadWriteCloser
	rest  *bytes.Buffer
}

func (c *CipherReadWriteCloser) Read(p []byte) (n int, err error) {
	n, err = c.conn.Read(p)
	copy(p, c.crypt.Decrypt(p[:n]))
	return
}

func (c *CipherReadWriteCloser) Write(p []byte) (n int, err error) {
	encrypted := c.crypt.Encrypt(p)

	curN := 0
	for len(encrypted) > 0 {
		curN, err = c.conn.Write(encrypted)
		n += curN
		if err != nil {
			return
		}
		encrypted = encrypted[n:]
	}

	return
}

func (c *CipherReadWriteCloser) Close() error {
	return c.conn.Close()
}

func newCipherReadWriteCloser(conn io.ReadWriteCloser, crypt Cipher) *CipherReadWriteCloser {
	return &CipherReadWriteCloser{
		conn:  conn,
		crypt: crypt,
		rest:  &bytes.Buffer{},
	}
}
