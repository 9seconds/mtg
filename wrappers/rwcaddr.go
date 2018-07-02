package wrappers

import (
	"io"
	"net"
)

type ReadWriteCloserWithAddr interface {
	io.ReadWriteCloser

	Addr() *net.TCPAddr
}
