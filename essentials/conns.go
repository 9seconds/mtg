package essentials

import "net"

type CloseableReader interface {
	CloseRead() error
}

type CloseableWriter interface {
	CloseWrite() error
}

type Conn interface {
	net.Conn
	CloseableReader
	CloseableWriter
}
