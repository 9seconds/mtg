package wrappers

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"net"

	"github.com/9seconds/mtg/utils"
	"github.com/juju/errors"
)

type BlockCipher struct {
	buf *bytes.Buffer

	conn      WrapStreamReadWriteCloser
	encryptor cipher.BlockMode
	decryptor cipher.BlockMode
}

func (b *BlockCipher) Read(p []byte) (int, error) {
	if b.buf.Len() > 0 {
		return b.flush(p)
	}

	buf := []byte{}
	for len(buf) == 0 || len(buf)%aes.BlockSize != 0 {
		rv, err := utils.ReadCurrentData(b.conn)
		if err != nil {
			return 0, errors.Annotate(err, "Cannot read from socket")
		}
		buf = append(buf, rv...)
	}

	b.decryptor.CryptBlocks(buf, buf)
	b.buf.Write(buf)

	return b.flush(p)
}

func (b *BlockCipher) flush(p []byte) (int, error) {
	if b.buf.Len() <= len(p) {
		sizeToReturn := b.buf.Len()
		copy(p, b.buf.Bytes())
		b.buf.Reset()
		return sizeToReturn, nil
	}

	return b.buf.Read(p)
}

func (b *BlockCipher) Write(p []byte) (int, error) {
	if len(p)%aes.BlockSize > 0 {
		return 0, errors.Errorf("Incorrect block size %d", len(p))
	}

	encrypted := make([]byte, len(p))
	b.encryptor.CryptBlocks(encrypted, p)

	return b.conn.Write(encrypted)
}

func (b *BlockCipher) LogDebug(msg string, data ...interface{}) {
	b.conn.LogDebug(msg, data...)
}

func (b *BlockCipher) LogInfo(msg string, data ...interface{}) {
	b.conn.LogInfo(msg, data...)
}

func (b *BlockCipher) LogWarn(msg string, data ...interface{}) {
	b.conn.LogWarn(msg, data...)
}

func (b *BlockCipher) LogError(msg string, data ...interface{}) {
	b.conn.LogError(msg, data...)
}

func (b *BlockCipher) LocalAddr() *net.TCPAddr {
	return b.conn.LocalAddr()
}

func (b *BlockCipher) RemoteAddr() *net.TCPAddr {
	return b.conn.RemoteAddr()
}

func (b *BlockCipher) Close() error {
	return b.conn.Close()
}

func NewBlockCipher(conn WrapStreamReadWriteCloser, encryptor, decryptor cipher.BlockMode) WrapStreamReadWriteCloser {
	return &BlockCipher{
		buf:       &bytes.Buffer{},
		conn:      conn,
		encryptor: encryptor,
		decryptor: decryptor,
	}
}
