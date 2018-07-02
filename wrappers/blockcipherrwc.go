package wrappers

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"io"

	"github.com/juju/errors"
)

type BlockCipherReadWriteCloser struct {
	buf *bytes.Buffer

	conn      io.ReadWriteCloser
	encryptor cipher.BlockMode
	decryptor cipher.BlockMode
}

func (c *BlockCipherReadWriteCloser) Read(p []byte) (int, error) {
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

	return c.flush(p)
}

func (c *BlockCipherReadWriteCloser) Write(p []byte) (int, error) {
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

func (c *BlockCipherReadWriteCloser) Close() error {
	defer putBuffer(c.buf)
	return c.conn.Close()
}

func (c *BlockCipherReadWriteCloser) flush(p []byte) (int, error) {
	sizeToRead := len(p)
	if c.buf.Len() < sizeToRead {
		sizeToRead = c.buf.Len()
	}
	sizeToRead = aes.BlockSize * (sizeToRead / aes.BlockSize)

	c.decryptor.CryptBlocks(p, c.buf.Bytes()[:sizeToRead])
	if sizeToRead == c.buf.Len() {
		c.buf.Reset()
	} else {
		leftover := c.buf.Bytes()[sizeToRead:]
		putBuffer(c.buf)
		c.buf = getBuffer()
		c.buf.Write(leftover)
	}

	return sizeToRead, nil
}

func NewBlockCipherRWC(conn io.ReadWriteCloser, encryptor, decryptor cipher.BlockMode) io.ReadWriteCloser {
	return &BlockCipherReadWriteCloser{
		buf:       getBuffer(),
		conn:      conn,
		encryptor: encryptor,
		decryptor: decryptor,
	}
}
