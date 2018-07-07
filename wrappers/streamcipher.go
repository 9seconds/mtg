package wrappers

import (
	"crypto/cipher"
	"net"

	"github.com/juju/errors"
)

type StreamCipher struct {
	encryptor cipher.Stream
	decryptor cipher.Stream
	conn      WrapStreamReadWriteCloser
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

func (s *StreamCipher) LogDebug(msg string, data ...interface{}) {
	s.conn.LogDebug(msg, data...)
}

func (s *StreamCipher) LogInfo(msg string, data ...interface{}) {
	s.conn.LogInfo(msg, data...)
}

func (s *StreamCipher) LogWarn(msg string, data ...interface{}) {
	s.conn.LogWarn(msg, data...)
}

func (s *StreamCipher) LogError(msg string, data ...interface{}) {
	s.conn.LogError(msg, data...)
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

func NewStreamCipher(conn WrapStreamReadWriteCloser, encryptor, decryptor cipher.Stream) WrapStreamReadWriteCloser {
	return &StreamCipher{
		conn:      conn,
		encryptor: encryptor,
		decryptor: decryptor,
	}
}
