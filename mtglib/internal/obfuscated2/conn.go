package obfuscated2

import (
	"crypto/cipher"
	"net"
)

type Conn struct {
	net.Conn

	Encryptor cipher.Stream
	Decryptor cipher.Stream

	writeBuf []byte
}

func (c *Conn) Read(p []byte) (int, error) {
	n, err := c.Conn.Read(p)
	if err != nil {
		return n, err // nolint: wrapcheck
	}

	c.Decryptor.XORKeyStream(p, p[:n])

	return n, nil
}

func (c *Conn) Write(p []byte) (int, error) {
	c.writeBuf = append(c.writeBuf[:0], p...)
	c.Encryptor.XORKeyStream(c.writeBuf, p)

	return c.Conn.Write(c.writeBuf)
}
