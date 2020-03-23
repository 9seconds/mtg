package packet

import (
	"bytes"
	"sync"
)

var (
	poolMtprotoFrameBytesBuffer = sync.Pool{
		New: func() interface{} {
			return &bytes.Buffer{}
		},
	}
)

func acquireMtprotoFrameBytesBuffer() *bytes.Buffer {
	return poolMtprotoFrameBytesBuffer.Get().(*bytes.Buffer)
}

func releaseMtprotoFrameBytesBuffer(buf *bytes.Buffer) {
	buf.Reset()
	poolMtprotoFrameBytesBuffer.Put(buf)
}
