package rwc

import (
	"crypto/cipher"
	"net"

	"github.com/juju/errors"
	"go.uber.org/zap"

	"github.com/9seconds/mtg/wrappers"
)

// StreamCipher is a wrapper which encrypts/decrypts stream with AES-CTR
// (as a part of obfuscated2 protocol).
type StreamCipher struct {
	encryptor cipher.Stream
	decryptor cipher.Stream
	conn      wrappers.StreamReadWriteCloser
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
	encrypted := make([]byte, len(p))
	s.encryptor.XORKeyStream(encrypted, p)

	return s.conn.Write(encrypted)
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

func (s *StreamCipher) SocketID() string {
	return s.conn.SocketID()
}

// NewStreamCipher creates new stream cipher wrapper.
func NewStreamCipher(conn wrappers.StreamReadWriteCloser, encryptor, decryptor cipher.Stream) wrappers.StreamReadWriteCloser {
	return &StreamCipher{
		conn:      conn,
		logger:    conn.Logger().Named("stream-cipher"),
		encryptor: encryptor,
		decryptor: decryptor,
	}
}
