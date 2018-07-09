package wrappers

import (
	"crypto/cipher"
	"net"

	"github.com/juju/errors"
	"go.uber.org/zap"
)

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
	encrypted := make([]byte, len(p))
	s.encryptor.XORKeyStream(encrypted, p)

	return s.conn.Write(encrypted)
}

func (s *StreamCipher) Logger() *zap.SugaredLogger {
	return s.logger
}

func (s *StreamCipher) LocalAddr() *net.TCPAddr {
	return s.conn.LocalAddr()
}

func (s *StreamCipher) RemoteAddr() *net.TCPAddr {
	return s.conn.RemoteAddr()
}

func (s *StreamCipher) Close() error {
	return s.conn.Close()
}

func NewStreamCipher(conn StreamReadWriteCloser, encryptor, decryptor cipher.Stream) StreamReadWriteCloser {
	return &StreamCipher{
		conn:      conn,
		logger:    conn.Logger().Named("stream-cipher"),
		encryptor: encryptor,
		decryptor: decryptor,
	}
}
