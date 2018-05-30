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
	c.rest.Write(c.crypt.Encrypt(p))
	newP := c.rest.Bytes()[:len(p)]
	n, err = c.conn.Write(newP)
	c.rest = bytes.NewBuffer(c.rest.Bytes()[n:])
	return
}

func (c *CipherReadWriteCloser) Close() error {
	var err1 error
	if c.rest.Len() > 0 {
		_, err1 = c.conn.Write(c.rest.Bytes())
	}
	err2 := c.conn.Close()

	if err2 != nil {
		return err2
	}
	return err1
}

func newCipherReadWriteCloser(conn io.ReadWriteCloser, crypt Cipher) io.ReadWriteCloser {
	return &CipherReadWriteCloser{
		conn:  conn,
		crypt: crypt,
		rest:  &bytes.Buffer{},
	}
}
