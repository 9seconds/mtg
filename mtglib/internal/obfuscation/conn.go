package obfuscation

import (
	"crypto/cipher"

	"github.com/9seconds/mtg/v2/essentials"
)

type conn struct {
	essentials.Conn

	sendCipher cipher.Stream
	recvCipher cipher.Stream
}

func (c conn) Read(p []byte) (int, error) {
	n, err := c.Conn.Read(p)
	if err != nil {
		return n, err
	}

	c.recvCipher.XORKeyStream(p, p[:n])

	return n, nil
}

func (c conn) Write(p []byte) (int, error) {
	// yes, this is a bit violent and goes against a contract in io.Writer
	// but we do it to avoid creating a new buffer just to perform this
	// encryption.
	c.sendCipher.XORKeyStream(p, p)

	return c.Conn.Write(p)
}
