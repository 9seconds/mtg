package proxy

import (
	"io"
	"net"
	"time"
)

// TimeoutReadWriteCloser sets timeouts for read/write into underlying
// network connection.
type TimeoutReadWriteCloser struct {
	conn         net.Conn
	readTimeout  time.Duration
	writeTimeout time.Duration
}

// Read reads from connection
func (t *TimeoutReadWriteCloser) Read(p []byte) (int, error) {
	t.conn.SetReadDeadline(time.Now().Add(t.readTimeout)) // nolint: errcheck, gas
	return t.conn.Read(p)
}

// Write writes into connection.
func (t *TimeoutReadWriteCloser) Write(p []byte) (int, error) {
	t.conn.SetWriteDeadline(time.Now().Add(t.writeTimeout)) // nolint: errcheck, gas
	return t.conn.Write(p)
}

// Close closes underlying connection.
func (t *TimeoutReadWriteCloser) Close() error {
	return t.conn.Close()
}

func newTimeoutReadWriteCloser(conn net.Conn, readTimeout, writeTimeout time.Duration) io.ReadWriteCloser {
	return &TimeoutReadWriteCloser{
		conn:         conn,
		readTimeout:  readTimeout,
		writeTimeout: writeTimeout,
	}
}
