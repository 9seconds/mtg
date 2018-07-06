package wrappers

import (
	"bytes"

	"github.com/juju/errors"
)

type BufferedReader struct {
	Buffer *bytes.Buffer
}

var (
	BufferedReaderContinue = errors.New("Please continue reading")
)

func (b *BufferedReader) BufferedRead(p []byte, callback func() error) (int, error) {
	if b.Buffer.Len() > 0 {
		return b.flush(p)
	}
	if err := callback(); err != nil {
		return 0, err
	}
	return b.flush(p)
}

func (b *BufferedReader) flush(p []byte) (int, error) {
	if b.Buffer.Len() <= len(p) {
		sizeToReturn := b.Buffer.Len()
		copy(p, b.Buffer.Bytes())
		b.Buffer.Reset()
		return sizeToReturn, nil
	}

	return b.Buffer.Read(p)
}

func NewBufferedReader() BufferedReader {
	return BufferedReader{Buffer: &bytes.Buffer{}}
}
