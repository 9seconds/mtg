package tlstypes

import (
	"bytes"
	"sync"
)

var (
	poolBytesBuffer = sync.Pool{
		New: func() interface{} {
			return &bytes.Buffer{}
		},
	}
)

func acquireBytesBuffer() *bytes.Buffer {
	return poolBytesBuffer.Get().(*bytes.Buffer)
}

func releaseBytesBuffer(buf *bytes.Buffer) {
	buf.Reset()
	poolBytesBuffer.Put(buf)
}
