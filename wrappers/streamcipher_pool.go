package wrappers

import (
	"bytes"
	"sync"
)

var (
	streamCipherBufferPool = sync.Pool{
		New: func() interface{} {
			return &bytes.Buffer{}
		},
	}
)
