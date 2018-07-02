package mtproto

import (
	"bytes"
	"io"
)

type BytesRWC interface {
	Write(*bytes.Buffer) (int, error)
	Read([]byte) (int, error)
	Close() error
}

type StartBytesRWC struct {
	conn BytesRWC
}

func (s *StartBytesRWC) Write(p []byte) (int, error) {
	buf := GetBuffer()
	buf.Write(p)
	defer ReturnBuffer(buf)

	return s.conn.Write(buf)
}

func (s *StartBytesRWC) Read(p []byte) (int, error) {
	return s.conn.Read(p)
}

func (s *StartBytesRWC) Close() error {
	return s.conn.Close()
}

type FinishBytesRWC struct {
	conn io.ReadWriteCloser
}

func (f *FinishBytesRWC) Write(buf *bytes.Buffer) (int, error) {
	return f.conn.Write(buf.Bytes())
}

func (f *FinishBytesRWC) Read(p []byte) (int, error) {
	return f.conn.Read(p)
}

func (f *FinishBytesRWC) Close() error {
	return f.conn.Close()
}
