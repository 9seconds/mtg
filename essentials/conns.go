package essentials

import (
	"io"
	"net"
)

// CloseableReader is an [io.Reader] interface that can close its reading end.
type CloseableReader interface {
	io.Reader
	CloseRead() error
}

// CloseableWriter is an [io.Writer] that can close its writing end.
type CloseableWriter interface {
	io.Writer
	CloseWrite() error
}

// Conn is an extension of [net.Conn] that can close its ends. This mostly
// implies TCP connections.
type Conn interface {
	net.Conn
	CloseableReader
	CloseableWriter
}
