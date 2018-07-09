package wrappers

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"net"

	"go.uber.org/zap"

	"github.com/9seconds/mtg/utils"
	"github.com/juju/errors"
)

// BlockCipher is a stream writer which encrypts/decrypts blocks of data
// with AES CBC. This also is buffered reader. It means, that block
// reading is transparent for it, you can assume you are working with
// good old io.Reader.
type BlockCipher struct {
	buf *bytes.Buffer

	logger    *zap.SugaredLogger
	conn      StreamReadWriteCloser
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

// Logger returns an instance of the logger for this wrapper.
func (b *BlockCipher) Logger() *zap.SugaredLogger {
	return b.logger
}

// LocalAddr returns local address of the underlying net.Conn.
func (b *BlockCipher) LocalAddr() *net.TCPAddr {
	return b.conn.LocalAddr()
}

// RemoteAddr returns remote address of the underlying net.Conn.
func (b *BlockCipher) RemoteAddr() *net.TCPAddr {
	return b.conn.RemoteAddr()
}

// Close closes underlying net.Conn.
func (b *BlockCipher) Close() error {
	return b.conn.Close()
}

// NewBlockCipher creates new instance of BlockCipher based on given data.
func NewBlockCipher(conn StreamReadWriteCloser, encryptor, decryptor cipher.BlockMode) StreamReadWriteCloser {
	return &BlockCipher{
		buf:       &bytes.Buffer{},
		conn:      conn,
		logger:    conn.Logger().Named("block-cipher"),
		encryptor: encryptor,
		decryptor: decryptor,
	}
}
