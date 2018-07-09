package wrappers

import (
	"io"
	"net"

	"go.uber.org/zap"
)

// Wrap is a base interface for all wrappers in this package.
type Wrap interface {
	Logger() *zap.SugaredLogger
	LocalAddr() *net.TCPAddr
	RemoteAddr() *net.TCPAddr
}

// Writer is a base interface for writers of this package.
type Writer interface {
	io.Writer
	Wrap
}

// Closer is a base interface for wrappers of this package which can
// close connections.
type Closer interface {
	io.Closer
	Wrap
}

// WriteCloser is a base interface for wrappers of this package which
// can write to and close connections.
type WriteCloser interface {
	io.Closer
	Writer
}

// StreamReader is a base interface for wrappers which can read from the
// stream.
type StreamReader interface {
	io.Reader
	Wrap
}

// StreamReadCloser is a base interface for wrappers which can read from
// and close the connections.
type StreamReadCloser interface {
	io.Closer
	StreamReader
}

// StreamReadWriter is a base interface for wrappers which can read from
// and write to the connections.
type StreamReadWriter interface {
	io.Writer
	StreamReader
}

// StreamWriteCloser is a base interface for wrappers which can write to
// and close the connections.
type StreamWriteCloser interface {
	io.WriteCloser
	Wrap
}

// StreamReadWriteCloser is a base interface for stream processors.
type StreamReadWriteCloser interface {
	io.Closer
	StreamReadWriter
}

// PacketReader is a base interface for wrappers which reads 'packets'.
// packets are atoms so you either get a packet or you get an error You
// cannot resume reading from packet.
type PacketReader interface {
	Read() ([]byte, error)
	Wrap
}

// PacketWriter is a base interface for wrappers which can write packets.
type PacketWriter interface {
	io.Writer
	Wrap
}

// PacketReadWriter is a base interface for wrappers which can read from
// and write packets.
type PacketReadWriter interface {
	io.Writer
	PacketReader
}

// PacketReadCloser is a base interface for wrappers which can read
// packets and close the connection.
type PacketReadCloser interface {
	io.Closer
	PacketReader
}

// PacketWriteCloser is a base interface for wrappers which can write
// packets and close the connection.
type PacketWriteCloser interface {
	io.Writer
	io.Closer
	Wrap
}

// PacketReadWriteCloser is a base interface for packet processors.
type PacketReadWriteCloser interface {
	io.Closer
	PacketReadWriter
}
