package relay

import "sync"

var copyBufferPool = sync.Pool{
	New: func() interface{} {
		rv := make([]byte, copyBufferSize)

		return &rv
	},
}

func acquireCopyBuffer() *[]byte {
	return copyBufferPool.Get().(*[]byte)
}

func releaseCopyBuffer(buf *[]byte) {
	copyBufferPool.Put(buf)
}
