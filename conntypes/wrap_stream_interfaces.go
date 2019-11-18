package conntypes

import (
	"io"
	"time"
)

type BaseStreamReaderWithTimeout interface {
	ReadTimeout([]byte, time.Duration) (int, error)
}

type BaseStreamWriterWithTimeout interface {
	WriteTimeout([]byte, time.Duration) (int, error)
}

type StreamReader interface {
	Wrap
	io.Reader
	BaseStreamReaderWithTimeout
}

type StreamWriter interface {
	Wrap
	io.Writer
	BaseStreamWriterWithTimeout
}

type StreamCloser interface {
	Wrap
	io.Closer
}

type StreamReadCloser interface {
	Wrap
	io.ReadCloser
	BaseStreamReaderWithTimeout
}

type StreamWriteCloser interface {
	Wrap
	io.WriteCloser
	BaseStreamWriterWithTimeout
}

type StreamReadWriter interface {
	Wrap
	io.ReadWriter
	BaseStreamReaderWithTimeout
}

type StreamReadWriteCloser interface {
	Wrap
	io.ReadWriteCloser
	BaseStreamReaderWithTimeout
	BaseStreamWriterWithTimeout
}
