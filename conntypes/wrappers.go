package conntypes

import (
	"io"
	"net"
	"time"

	"go.uber.org/zap"
)

// Wrap is a base interface for all wrappers in this package.
type Wrap interface {
	Conn() net.Conn
	Logger() *zap.SugaredLogger
	LocalAddr() *net.TCPAddr
	RemoteAddr() *net.TCPAddr
}

type BaseReaderWithTimeout interface {
	ReadTimeout([]byte, time.Duration) (int, error)
}

type BaseWriterWithTimeout interface {
	WriteTimeout([]byte, time.Duration) (int, error)
}

type BasePacketReader interface {
	Read() (Packet, error)
}

type BasePacketWriter interface {
	Write(Packet) error
}

type StreamReader interface {
	Wrap
	io.Reader
	BaseReaderWithTimeout
}

type StreamWriter interface {
	Wrap
	io.Writer
	BaseWriterWithTimeout
}

type StreamCloser interface {
	Wrap
	io.Closer
}

type StreamReadCloser interface {
	Wrap
	io.ReadCloser
	BaseReaderWithTimeout
}

type StreamWriteCloser interface {
	Wrap
	io.WriteCloser
	BaseWriterWithTimeout
}

type StreamReadWriter interface {
	Wrap
	io.ReadWriter
	BaseReaderWithTimeout
}

type StreamReadWriteCloser interface {
	Wrap
	io.ReadWriteCloser
	BaseReaderWithTimeout
	BaseWriterWithTimeout
}

type PacketReader interface {
	Wrap
	BasePacketReader
}

type PacketWriter interface {
	Wrap
	BasePacketWriter
}

type PacketCloser interface {
	Wrap
	io.Closer
}

type PacketReadCloser interface {
	Wrap
	BasePacketReader
	io.Closer
}

type PacketWriteCloser interface {
	Wrap
	BasePacketWriter
	io.Closer
}

type PacketReadWriter interface {
	Wrap
	BasePacketWriter
	BasePacketReader
}

type PacketReadWriteCloser interface {
	Wrap
	BasePacketWriter
	BasePacketReader
	io.Closer
}
