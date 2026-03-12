package fake

import (
	"bytes"
	"sync"
)

var bytesPool = sync.Pool{
	New: func() any {
		return &bytes.Buffer{}
	},
}

func acquireBuffer() *bytes.Buffer {
	return bytesPool.Get().(*bytes.Buffer)
}

func releaseBuffer(b *bytes.Buffer) {
	b.Reset()
	bytesPool.Put(b)
}
