package wrappers

import (
	"bytes"
	"crypto/cipher"
	"io"

	"github.com/juju/errors"
)

type BlockCipherReadWriteCloser struct {
	encryptor cipher.BlockMode
	decryptor cipher.BlockMode
	conn      io.ReadWriteCloser
	buf       *bytes.Buffer
}

func (c *BlockCipherReadWriteCloser) Read(p []byte) (n int, err error) {
	blockSize := c.decryptor.BlockSize()
	if len(p) < blockSize {
		return 0, errors.New("Cannot read less than blocksize")
	}

	n, err = c.conn.Read(p)
	c.buf.Write(p[:n])

	wantToRead := c.getFullBlocks(len(p), blockSize)
	haveBlocks := c.getFullBlocks(c.buf.Len(), blockSize)
	if haveBlocks < wantToRead {
		wantToRead = haveBlocks
	}
	wantToRead *= blockSize

	data := c.buf.Bytes()
	c.decryptor.CryptBlocks(p, data[:wantToRead])
	c.buf = bytes.NewBuffer(data[wantToRead:])

	return wantToRead, err
}

func (c *BlockCipherReadWriteCloser) Write(p []byte) (n int, err error) {
	blockSize := c.encryptor.BlockSize()
	if len(p)%blockSize != 0 {
		return 0, errors.New("Write size should be compatible with block size")
	}

	buf := make([]byte, len(p))
	c.encryptor.CryptBlocks(buf, p)

	return c.conn.Write(buf)
}

func (c *BlockCipherReadWriteCloser) Close() error {
	return c.conn.Close()
}

func (c *BlockCipherReadWriteCloser) getFullBlocks(number, blockSize int) int {
	blocks := number / blockSize

	if blocks > 0 && number%blockSize != 0 {
		blocks--
	}

	return blocks
}

func NewBlockCipherRWC(conn io.ReadWriteCloser, encryptor, decryptor cipher.BlockMode) io.ReadWriteCloser {
	return &BlockCipherReadWriteCloser{
		conn:      conn,
		encryptor: encryptor,
		decryptor: decryptor,
	}
}
