//go:build !linux
// +build !linux

package relay

import "sync"

const (
	copyBufferSize = 64 * 1024
)

var copyBufferPool = sync.Pool{
	New: func() interface{} {
		rv := make([]byte, copyBufferSize)

		return &rv
	},
}

func acquireCopyBuffer() *[]byte {
	return copyBufferPool.Get().(*[]byte) //nolint: forcetypeassert
}

func releaseCopyBuffer(buf *[]byte) {
	copyBufferPool.Put(buf)
}
