package proxy

import "sync"

const copyBufferSize = 30 * 1024

var copyPool sync.Pool

func init() {
	copyPool = sync.Pool{
		New: func() interface{} {
			data := make([]byte, copyBufferSize)
			return &data
		},
	}
}
