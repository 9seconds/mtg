package faketls

import (
	"bytes"
	"sync"
)

var bytesBufferPool = sync.Pool{
	New: func() interface{} {
		return &bytes.Buffer{}
	},
}

func acquireBytesBuffer() *bytes.Buffer {
	return bytesBufferPool.Get().(*bytes.Buffer) //nolint: forcetypeassert
}

func releaseBytesBuffer(b *bytes.Buffer) {
	b.Reset()
	bytesBufferPool.Put(b)
}
