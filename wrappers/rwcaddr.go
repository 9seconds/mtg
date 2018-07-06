package wrappers

import (
	"io"
	"net"
)

type ReadWriteCloserWithAddr interface {
	io.ReadWriteCloser

	LocalAddr() *net.TCPAddr
	RemoteAddr() *net.TCPAddr
	SocketID() string
}
