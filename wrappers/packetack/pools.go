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
	poolProxyBytesBuffer = sync.Pool{
		New: func() interface{} {
			return &bytes.Buffer{}
		},
	}
)

func acquireClientBytesBuffer() *bytes.Buffer {
	return poolClientBytesBuffer.Get().(*bytes.Buffer)
}

func acquireProxyBytesBuffer() *bytes.Buffer {
	return poolProxyBytesBuffer.Get().(*bytes.Buffer)
}

func releaseClientBytesBuffer(buf *bytes.Buffer) {
	buf.Reset()
	poolClientBytesBuffer.Put(buf)
}

func releaseProxyBytesBuffer(buf *bytes.Buffer) {
	buf.Reset()
	poolProxyBytesBuffer.Put(buf)
}
