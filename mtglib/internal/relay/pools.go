package relay

import "sync"

var bufPool = sync.Pool{
	New: func() any {
		b := make([]byte, bufPoolSize)
		return &b
	},
}

func acquireBuffer() *[]byte {
	return bufPool.Get().(*[]byte)
}

func releaseBuffer(p *[]byte) {
	bufPool.Put(p)
}
