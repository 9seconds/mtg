package wrappers

import (
	"bytes"
	"sync"
)

var bufPool sync.Pool

func getBuffer() *bytes.Buffer {
	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()

	return buf
}

func putBuffer(buf *bytes.Buffer) {
	bufPool.Put(buf)
}

func init() {
	bufPool = sync.Pool{
		New: func() interface{} {
			return &bytes.Buffer{}
		},
	}
}
