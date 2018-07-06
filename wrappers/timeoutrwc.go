package wrappers

import (
	"net"
	"time"

	"github.com/9seconds/mtg/config"
)

type TimeoutReadWriteCloserWithAddr struct {
	conn       net.Conn
	sock       string
	publicIPv4 net.IP
	publicIPv6 net.IP
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

func (t *TimeoutReadWriteCloserWithAddr) RemoteAddr() *net.TCPAddr {
	return t.conn.RemoteAddr().(*net.TCPAddr)
}

func (t *TimeoutReadWriteCloserWithAddr) LocalAddr() *net.TCPAddr {
	addr := t.conn.LocalAddr().(*net.TCPAddr)
	newAddr := *addr

	if t.RemoteAddr().IP.To4() != nil {
		if t.publicIPv4 != nil {
			newAddr.IP = t.publicIPv4
		}
	} else if t.publicIPv6 != nil {
		newAddr.IP = t.publicIPv6
	}

	return &newAddr
}

func (t *TimeoutReadWriteCloserWithAddr) SocketID() string {
	return t.sock
}

func NewTimeoutRWC(conn net.Conn, sock string, ipv4, ipv6 net.IP) ReadWriteCloserWithAddr {
	return &TimeoutReadWriteCloserWithAddr{
		conn:       conn,
		publicIPv4: ipv4,
		publicIPv6: ipv6,
		sock:       sock,
	}
}
