package proxy

import (
	"bytes"
	"io"
)

// Cipher is an interface to anything which can encrypt and decrypt
type Cipher interface {
	Encrypt([]byte) []byte
	Decrypt([]byte) []byte
}

// CipherReadWriteCloser wraps connection for transparent encryption
type CipherReadWriteCloser struct {
	crypt Cipher
	conn  io.ReadWriteCloser
	rest  *bytes.Buffer
}

// Read reads from connection
func (c *CipherReadWriteCloser) Read(p []byte) (n int, err error) {
	n, err = c.conn.Read(p)
	copy(p, c.crypt.Decrypt(p[:n]))
	return
}

// Write writes into connection.
func (c *CipherReadWriteCloser) Write(p []byte) (int, error) {
	encrypted := c.crypt.Encrypt(p)
	allWritten := 0

	for len(encrypted) > 0 {
		n, err := c.conn.Write(encrypted)
		allWritten += n
		if err != nil {
			return allWritten, err
		}
		encrypted = encrypted[n:]
	}

	return allWritten, nil
}

// Close closes underlying connection.
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
