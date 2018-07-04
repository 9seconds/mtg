package wrappers

import (
	"crypto/aes"
	"crypto/cipher"
	"net"

	"github.com/juju/errors"
)

type BlockCipherReadWriteCloserWithAddr struct {
	BufferedReader

	conn      ReadWriteCloserWithAddr
	encryptor cipher.BlockMode
	decryptor cipher.BlockMode
}

func (c *BlockCipherReadWriteCloserWithAddr) Read(p []byte) (int, error) {
	return c.BufferedRead(p, func() error {
		bufferLength := c.Buffer.Len()
		for bufferLength%aes.BlockSize != 0 || bufferLength == 0 {
			n, err := c.conn.Read(p)
			if err != nil {
				return errors.Annotate(err, "Cannot read from socket")
			}
			c.Buffer.Write(p[:n])
			bufferLength = c.Buffer.Len()
		}
		c.decryptor.CryptBlocks(c.Buffer.Bytes(), c.Buffer.Bytes())

		return nil
	})
}

func (c *BlockCipherReadWriteCloserWithAddr) Write(p []byte) (int, error) {
	if len(p)%aes.BlockSize > 0 {
		return 0, errors.Errorf("Incorrect block size %d", len(p))
	}

	encrypted := make([]byte, len(p))
	c.encryptor.CryptBlocks(encrypted, p)

	return c.conn.Write(encrypted)
}

func (c *BlockCipherReadWriteCloserWithAddr) Close() error {
	return c.conn.Close()
}

func (c *BlockCipherReadWriteCloserWithAddr) LocalAddr() *net.TCPAddr {
	return c.conn.LocalAddr()
}

func (c *BlockCipherReadWriteCloserWithAddr) RemoteAddr() *net.TCPAddr {
	return c.conn.RemoteAddr()
}

func NewBlockCipherRWC(conn ReadWriteCloserWithAddr, encryptor, decryptor cipher.BlockMode) ReadWriteCloserWithAddr {
	return &BlockCipherReadWriteCloserWithAddr{
		BufferedReader: NewBufferedReader(),
		conn:           conn,
		encryptor:      encryptor,
		decryptor:      decryptor,
	}
}
