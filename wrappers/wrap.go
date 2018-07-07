package wrappers

import (
	"io"
	"net"
)

type Wrap interface {
	LogDebug(msg string, data ...interface{})
	LogInfo(msg string, data ...interface{})
	LogWarn(msg string, data ...interface{})
	LogError(msg string, data ...interface{})

	LocalAddr() *net.TCPAddr
	RemoteAddr() *net.TCPAddr
}

type WrapWriter interface {
	io.Writer
	Wrap
}

type WrapWriteCloser interface {
	io.Closer
	WrapWriter
}

type WrapStreamReader interface {
	io.Reader
	Wrap
}

type WrapStreamReadCloser interface {
	io.Closer
	WrapStreamReader
}

type WrapStreamReadWriter interface {
	io.Writer
	WrapStreamReader
}

type WrapStreamWriteCloser interface {
	io.Closer
	io.Writer
	Wrap
}

type WrapStreamReadWriteCloser interface {
	io.Closer
	WrapStreamReadWriter
}

type WrapPacketReader interface {
	Read() ([]byte, error)
	Wrap
}

type WrapPacketWriter interface {
	io.Writer
	Wrap
}

type WrapPacketReadWriter interface {
	io.Writer
	WrapPacketReader
}

type WrapBlockReadCloser interface {
	io.Closer
	WrapPacketReader
}

type WrapPacketWriteCloser interface {
	io.Writer
	io.Closer
	Wrap
}

type WrapPacketReadWriteCloser interface {
	io.Closer
	WrapPacketReadWriter
}
