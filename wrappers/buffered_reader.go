package wrappers

import "bytes"

type BufferedReader struct {
	Buffer *bytes.Buffer
}

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
	sizeToRead := len(p)
	if b.Buffer.Len() < sizeToRead {
		sizeToRead = b.Buffer.Len()
	}

	data := b.Buffer.Bytes()
	copy(p, data[:sizeToRead])
	if sizeToRead == b.Buffer.Len() {
		b.Buffer.Reset()
	} else {
		b.Buffer = bytes.NewBuffer(data[sizeToRead:])
	}

	return sizeToRead, nil
}

func NewBufferedReader() BufferedReader {
	return BufferedReader{Buffer: &bytes.Buffer{}}
}
