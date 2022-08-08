package obfuscated2

import (
	"crypto/cipher"

	"github.com/9seconds/mtg/v2/essentials"
)

type Conn struct {
	essentials.Conn

	Encryptor cipher.Stream
	Decryptor cipher.Stream
}

func (c Conn) Read(p []byte) (int, error) {
	n, err := c.Conn.Read(p)
	if err != nil {
		return n, err //nolint: wrapcheck
	}

	c.Decryptor.XORKeyStream(p, p[:n])

	return n, nil
}

func (c Conn) Write(p []byte) (int, error) {
	buf := acquireBytesBuffer()
	defer releaseBytesBuffer(buf)

	buf.Write(p)

	payload := buf.Bytes()
	c.Encryptor.XORKeyStream(payload, payload)

	return c.Conn.Write(payload) //nolint: wrapcheck
}
