package bufferpool

import (
	"bytes"
	"sync"
)

const bufferPoolSize = 4 * 1024

var bufferPool sync.Pool

func Get() *bytes.Buffer {
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()

	return buf
}

func Return(buf *bytes.Buffer) {
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
