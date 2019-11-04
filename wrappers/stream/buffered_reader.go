package stream

import (
	"bytes"
	"time"
)

type bufferedReaderReadFunc func() ([]byte, error)

type bufferedReader struct {
	buf      bytes.Buffer
	readFunc bufferedReaderReadFunc
}

func (b *bufferedReader) Read(p []byte) (int, error) {
	if b.buf.Len() > 0 {
		return b.flush(p)
	}

	res, err := b.readFunc()
	if err != nil {
		return 0, err
	}
	b.buf.Write(res)

	return b.flush(p)
}

func (b *bufferedReader) ReadTimeout(p []byte, _ time.Duration) (int, error) {
	return b.Read(p)
}

func (b *bufferedReader) flush(p []byte) (int, error) {
	if b.buf.Len() > len(p) {
		return b.buf.Read(p)
	}

	sizeToReturn := b.buf.Len()
	copy(p, b.buf.Bytes())
	b.buf.Reset()

	return sizeToReturn, nil
}
