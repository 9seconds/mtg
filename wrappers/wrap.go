package wrappers

import (
	"io"
	"net"

	"go.uber.org/zap"
)

type Wrap interface {
	Logger() *zap.SugaredLogger
	LocalAddr() *net.TCPAddr
	RemoteAddr() *net.TCPAddr
}

type Writer interface {
	io.Writer
	Wrap
}

type Closer interface {
	io.Closer
	Wrap
}

type WriteCloser interface {
	io.Closer
	Writer
}

type StreamReader interface {
	io.Reader
	Wrap
}

type StreamReadCloser interface {
	io.Closer
	StreamReader
}

type StreamReadWriter interface {
	io.Writer
	StreamReader
}

type StreamWriteCloser interface {
	io.WriteCloser
	Wrap
}

type StreamReadWriteCloser interface {
	io.Closer
	StreamReadWriter
}

type PacketReader interface {
	Read() ([]byte, error)
	Wrap
}

type PacketWriter interface {
	io.Writer
	Wrap
}

type PacketReadWriter interface {
	io.Writer
	PacketReader
}

type PacketReadCloser interface {
	io.Closer
	PacketReader
}

type PacketWriteCloser interface {
	io.Writer
	io.Closer
	Wrap
}

type PacketReadWriteCloser interface {
	io.Closer
	PacketReadWriter
}
