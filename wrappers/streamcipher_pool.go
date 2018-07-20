package wrappers

import (
	"bytes"
	"sync"
)

var streamCipherBufferPool sync.Pool

func init() {
	streamCipherBufferPool = sync.Pool{
		New: func() interface{} {
			return &bytes.Buffer{}
		},
	}
}
