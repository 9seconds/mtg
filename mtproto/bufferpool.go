package mtproto

import (
	"bytes"
	"sync"
)

const bufferPoolSize = 4 * 1024

var bufferPool sync.Pool

func GetBuffer() *bytes.Buffer {
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()

	return buf
}

func ReturnBuffer(buf *bytes.Buffer) {
	bufferPool.Put(buf)
}

func init() {
	bufferPool = sync.Pool{
		New: func() interface{} {
			buf := &bytes.Buffer{}
			buf.Grow(bufferPoolSize)

			return buf
		},
	}
}
