package wrappers

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"net"

	"github.com/juju/errors"
)

type BlockCipherReadWriteCloserWithAddr struct {
	buf *bytes.Buffer

	conn      ReadWriteCloserWithAddr
	encryptor cipher.BlockMode
	decryptor cipher.BlockMode
}

func (c *BlockCipherReadWriteCloserWithAddr) Read(p []byte) (int, error) {
	if c.buf.Len() > 0 {
		return c.flush(p)
	}

	for c.buf.Len() == 0 || c.buf.Len()%aes.BlockSize != 0 {
		n, err := c.conn.Read(p)
		if err != nil {
			return 0, errors.Annotate(err, "Cannot read from socket")
		}
		c.buf.Write(p[:n])
	}
	c.decryptor.CryptBlocks(c.buf.Bytes(), c.buf.Bytes())

	return c.flush(p)
}

func (c *BlockCipherReadWriteCloserWithAddr) Write(p []byte) (int, error) {
	if len(p)%aes.BlockSize > 0 {
		return 0, errors.Errorf("Incorrect block size %d", len(p))
	}

	buf := getBuffer()
	defer putBuffer(buf)
	buf.Grow(len(p))
	buf.Write(p)

	encrypted := buf.Bytes()
	c.encryptor.CryptBlocks(encrypted, p)

	return c.conn.Write(encrypted)
}

func (c *BlockCipherReadWriteCloserWithAddr) Close() error {
	defer putBuffer(c.buf)
	return c.conn.Close()
}

func (c *BlockCipherReadWriteCloserWithAddr) Addr() *net.TCPAddr {
	return c.conn.Addr()
}

func (c *BlockCipherReadWriteCloserWithAddr) flush(p []byte) (int, error) {
	sizeToRead := len(p)
	if c.buf.Len() < sizeToRead {
		sizeToRead = c.buf.Len()
	}

	data := c.buf.Bytes()
	copy(p, data[:sizeToRead])
	if sizeToRead == c.buf.Len() {
		c.buf.Reset()
	} else {
		newBuf := getBuffer()
		newBuf.Write(data[sizeToRead:])

		putBuffer(c.buf)
		c.buf = newBuf
	}

	return sizeToRead, nil
}

func NewBlockCipherRWC(conn ReadWriteCloserWithAddr, encryptor, decryptor cipher.BlockMode) ReadWriteCloserWithAddr {
	return &BlockCipherReadWriteCloserWithAddr{
		buf:       getBuffer(),
		conn:      conn,
		encryptor: encryptor,
		decryptor: decryptor,
	}
}
