package wrappers

import (
	"bytes"
	"crypto/cipher"
	"io"
)

type StreamCipherReadWriteCloser struct {
	encryptor cipher.Stream
	decryptor cipher.Stream
	conn      io.ReadWriteCloser
	rest      *bytes.Buffer
}

// Read reads from connection
func (c *StreamCipherReadWriteCloser) Read(p []byte) (n int, err error) {
	n, err = c.conn.Read(p)
	c.decryptor.XORKeyStream(p, p[:n])
	return
}

// Write writes into connection.
func (c *StreamCipherReadWriteCloser) Write(p []byte) (int, error) {
	encrypted := make([]byte, len(p))
	c.encryptor.XORKeyStream(encrypted, p)
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
func (c *StreamCipherReadWriteCloser) Close() error {
	return c.conn.Close()
}

func NewStreamCipherRWC(conn io.ReadWriteCloser, encryptor, decryptor cipher.Stream) io.ReadWriteCloser {
	return &StreamCipherReadWriteCloser{
		conn:      conn,
		encryptor: encryptor,
		decryptor: decryptor,
		rest:      &bytes.Buffer{},
	}
}
