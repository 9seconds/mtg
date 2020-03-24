package faketls

import (
	"bytes"
	"sync"
)

const cloakBufferSize = 1024

var (
	poolBytesBuffer = sync.Pool{
		New: func() interface{} {
			return &bytes.Buffer{}
		},
	}
	poolCloakBuffer = sync.Pool{
		New: func() interface{} {
			rv := make([]byte, cloakBufferSize)
			return &rv
		},
	}
)

func acquireBytesBuffer() *bytes.Buffer {
	return poolBytesBuffer.Get().(*bytes.Buffer)
}

func acquireCloakBuffer() *[]byte {
	return poolCloakBuffer.Get().(*[]byte)
}

func releaseBytesBuffer(buf *bytes.Buffer) {
	poolBytesBuffer.Put(buf)
}

func releaseCloakBuffer(buf *[]byte) {
	poolCloakBuffer.Put(buf)
}
