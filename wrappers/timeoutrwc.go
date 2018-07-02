package wrappers

import (
	"io"
	"net"
	"time"

	"github.com/9seconds/mtg/config"
)

type TimeoutReadWriteCloser struct {
	conn net.Conn
}

func (t *TimeoutReadWriteCloser) Read(p []byte) (int, error) {
	t.conn.SetReadDeadline(time.Now().Add(config.TimeoutRead))
	return t.conn.Read(p)
}

func (t *TimeoutReadWriteCloser) Write(p []byte) (int, error) {
	t.conn.SetWriteDeadline(time.Now().Add(config.TimeoutWrite))
	return t.conn.Write(p)
}

func (t *TimeoutReadWriteCloser) Close() error {
	return t.conn.Close()
}

func NewTimeoutRWC(conn net.Conn) io.ReadWriteCloser {
	return &TimeoutReadWriteCloser{conn}
}
