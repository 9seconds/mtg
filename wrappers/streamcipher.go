package wrappers

import (
	"bytes"
	"crypto/cipher"
	"net"

	"github.com/juju/errors"
	"go.uber.org/zap"
)

// StreamCipher is a wrapper which encrypts/decrypts stream with AES-CTR
// (as a part of obfuscated2 protocol).
type StreamCipher struct {
	encryptor cipher.Stream
	decryptor cipher.Stream
	conn      StreamReadWriteCloser
	logger    *zap.SugaredLogger
}

func (s *StreamCipher) Read(p []byte) (int, error) {
	n, err := s.conn.Read(p)
	if err != nil {
		return 0, errors.Annotate(err, "Cannot read stream ciphered data")
	}
	s.decryptor.XORKeyStream(p, p[:n])

	return n, nil
}

func (s *StreamCipher) Write(p []byte) (int, error) {
	buf := streamCipherBufferPool.Get().(*bytes.Buffer)
	defer streamCipherBufferPool.Put(buf)

	buf.Reset()
	buf.Grow(len(p))
	buf.Write(p)

	data := buf.Bytes()
	s.encryptor.XORKeyStream(data, data)

	return s.conn.Write(data)
}

// Logger returns an instance of the logger for this wrapper.
func (s *StreamCipher) Logger() *zap.SugaredLogger {
	return s.logger
}

// LocalAddr returns local address of the underlying net.Conn.
func (s *StreamCipher) LocalAddr() *net.TCPAddr {
	return s.conn.LocalAddr()
}

// RemoteAddr returns remote address of the underlying net.Conn.
func (s *StreamCipher) RemoteAddr() *net.TCPAddr {
	return s.conn.RemoteAddr()
}

// Close closes underlying net.Conn instance.
func (s *StreamCipher) Close() error {
	return s.conn.Close()
}

// NewStreamCipher creates new stream cipher wrapper.
func NewStreamCipher(conn StreamReadWriteCloser, encryptor, decryptor cipher.Stream) StreamReadWriteCloser {
	return &StreamCipher{
		conn:      conn,
		logger:    conn.Logger().Named("stream-cipher"),
		encryptor: encryptor,
		decryptor: decryptor,
	}
}
