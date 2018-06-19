package wrappers

import (
	"crypto/cipher"
	"io"
)

// StreamCipherReadWriteCloser is a ReadWriteCloser which ciphers
// incoming and outgoing data with givem cipher.Stream instances.
type StreamCipherReadWriteCloser struct {
	encryptor cipher.Stream
	decryptor cipher.Stream
	conn      io.ReadWriteCloser
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

// NewStreamCipherRWC returns wrapper which transparently
// encrypts/decrypts traffic with obfuscated2 protocol.
func NewStreamCipherRWC(conn io.ReadWriteCloser, encryptor, decryptor cipher.Stream) io.ReadWriteCloser {
	return &StreamCipherReadWriteCloser{
		conn:      conn,
		encryptor: encryptor,
		decryptor: decryptor,
	}
}
