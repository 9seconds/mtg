package wrappers

import (
	"crypto/cipher"
	"net"
)

// StreamCipherReadWriteCloser is a ReadWriteCloser which ciphers
// incoming and outgoing data with givem cipher.Stream instances.
type StreamCipherReadWriteCloserWithAddr struct {
	encryptor cipher.Stream
	decryptor cipher.Stream
	conn      ReadWriteCloserWithAddr
}

// Read reads from connection
func (c *StreamCipherReadWriteCloserWithAddr) Read(p []byte) (n int, err error) {
	n, err = c.conn.Read(p)
	c.decryptor.XORKeyStream(p, p[:n])
	return
}

// Write writes into connection.
func (c *StreamCipherReadWriteCloserWithAddr) Write(p []byte) (int, error) {
	// This is to decrease an amount of allocations. Unfortunately, escape
	// analysis in (at least Golang 1.10) is absolutely not perfect. For
	// example, it understands that we want to have a slice locally, right?
	// But since slice is effectively 2 ints + uintptr to [number]byte, the
	// most heavyweight part is placed in heap.
	buf := getBuffer()
	defer putBuffer(buf)
	buf.Grow(len(p))
	buf.Write(p)

	encrypted := buf.Bytes()
	c.encryptor.XORKeyStream(encrypted, p)

	return c.conn.Write(encrypted)
}

// Close closes underlying connection.
func (c *StreamCipherReadWriteCloserWithAddr) Close() error {
	return c.conn.Close()
}

func (c *StreamCipherReadWriteCloserWithAddr) LocalAddr() *net.TCPAddr {
	return c.conn.LocalAddr()
}

func (c *StreamCipherReadWriteCloserWithAddr) RemoteAddr() *net.TCPAddr {
	return c.conn.RemoteAddr()
}

func (c *StreamCipherReadWriteCloserWithAddr) SocketID() string {
	return c.conn.SocketID()
}

// NewStreamCipherRWC returns wrapper which transparently
// encrypts/decrypts traffic with obfuscated2 protocol.
func NewStreamCipherRWC(conn ReadWriteCloserWithAddr, encryptor, decryptor cipher.Stream) ReadWriteCloserWithAddr {
	return &StreamCipherReadWriteCloserWithAddr{
		conn:      conn,
		encryptor: encryptor,
		decryptor: decryptor,
	}
}
