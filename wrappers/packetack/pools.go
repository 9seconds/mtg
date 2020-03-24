package packetack

import (
	"bytes"
	"sync"
)

var (
	poolClientBytesBuffer = sync.Pool{
		New: func() interface{} {
			return &bytes.Buffer{}
		},
	}
)

func acquireClientBytesBuffer() *bytes.Buffer {
	return poolClientBytesBuffer.Get().(*bytes.Buffer)
}

func releaseClientBytesBuffer(buf *bytes.Buffer) {
	buf.Reset()
	poolClientBytesBuffer.Put(buf)
}
