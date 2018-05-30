package server

import (
	"io"
	"net"
	"time"
)

type TimeoutReadWriteCloser struct {
	conn         net.Conn
	readTimeout  time.Duration
	writeTimeout time.Duration
}

func (t *TimeoutReadWriteCloser) Read(p []byte) (int, error) {
	t.conn.SetReadDeadline(time.Now().Add(t.readTimeout))
	return t.conn.Read(p)
}

func (t *TimeoutReadWriteCloser) Write(p []byte) (int, error) {
	t.conn.SetWriteDeadline(time.Now().Add(t.writeTimeout))
	return t.conn.Write(p)
}

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
