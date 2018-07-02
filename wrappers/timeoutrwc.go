package wrappers

import (
	"net"
	"time"

	"github.com/9seconds/mtg/config"
)

type TimeoutReadWriteCloserWithAddr struct {
	conn net.Conn
}

func (t *TimeoutReadWriteCloserWithAddr) Read(p []byte) (int, error) {
	t.conn.SetReadDeadline(time.Now().Add(config.TimeoutRead))
	return t.conn.Read(p)
}

func (t *TimeoutReadWriteCloserWithAddr) Write(p []byte) (int, error) {
	t.conn.SetWriteDeadline(time.Now().Add(config.TimeoutWrite))
	return t.conn.Write(p)
}

func (t *TimeoutReadWriteCloserWithAddr) Close() error {
	return t.conn.Close()
}

func (t *TimeoutReadWriteCloserWithAddr) Addr() *net.TCPAddr {
	return t.conn.RemoteAddr().(*net.TCPAddr)
}

func NewTimeoutRWC(conn net.Conn) ReadWriteCloserWithAddr {
	return &TimeoutReadWriteCloserWithAddr{conn}
}
