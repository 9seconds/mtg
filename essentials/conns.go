package essentials

import (
	"fmt"
	"io"
	"net"
	"syscall"
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

type netConnWrapper struct {
	net.Conn
}

func (n netConnWrapper) CloseRead() error {
	if conn, ok := n.Conn.(CloseableReader); ok {
		return conn.CloseRead()
	}

	return n.Close()
}

func (n netConnWrapper) CloseWrite() error {
	if conn, ok := n.Conn.(CloseableWriter); ok {
		return conn.CloseWrite()
	}

	return n.Close()
}

// WrapConn wraps a generic [net.Conn] into Conn.
func WrapNetConn(conn net.Conn) Conn {
	return netConnWrapper{conn}
}

func SetTCPWindowClamp(conn net.Conn, value int) error {
	sysConn, ok := conn.(syscall.Conn)
	if !ok {
		return nil
	}

	rawConn, err := sysConn.SyscallConn()
	if err != nil {
		return fmt.Errorf("cannot get raw connection: %w", err)
	}

	return SetRawTCPWindowClamp(rawConn, value)
}
