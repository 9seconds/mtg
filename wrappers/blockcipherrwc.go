package wrappers

import (
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"net"

	"github.com/juju/errors"

	"github.com/9seconds/mtg/utils"
)

type BlockCipherReadWriteCloserWithAddr struct {
	BufferedReader

	conn      ReadWriteCloserWithAddr
	encryptor cipher.BlockMode
	decryptor cipher.BlockMode
}

func (c *BlockCipherReadWriteCloserWithAddr) Read(p []byte) (int, error) {
	return c.BufferedRead(p, func() error {
		var buf []byte

		for len(buf) == 0 || len(buf)%aes.BlockSize != 0 {
			rv, err := utils.ReadCurrentData(c.conn)
			if err != nil {
				return errors.Annotate(err, "Cannot read from socket")
			}
			buf = append(buf, rv...)
		}

		c.decryptor.CryptBlocks(buf, buf)
		c.Buffer.Write(buf)

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
	fmt.Println("BlockCipherReadWriteCloserWithAddr closes", "sockid", c.SocketID(), "bufsize", c.Buffer.Len())
	return c.conn.Close()
}

func (c *BlockCipherReadWriteCloserWithAddr) LocalAddr() *net.TCPAddr {
	return c.conn.LocalAddr()
}

func (c *BlockCipherReadWriteCloserWithAddr) RemoteAddr() *net.TCPAddr {
	return c.conn.RemoteAddr()
}

func (c *BlockCipherReadWriteCloserWithAddr) SocketID() string {
	return c.conn.SocketID()
}

func NewBlockCipherRWC(conn ReadWriteCloserWithAddr, encryptor, decryptor cipher.BlockMode) ReadWriteCloserWithAddr {
	return &BlockCipherReadWriteCloserWithAddr{
		BufferedReader: NewBufferedReader(),
		conn:           conn,
		encryptor:      encryptor,
		decryptor:      decryptor,
	}
}
