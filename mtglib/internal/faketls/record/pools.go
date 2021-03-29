package record

import (
	"bytes"
	"sync"
)

var (
	recordPool = sync.Pool{
		New: func() interface{} {
			return &Record{}
		},
	}
	bytesBufferPool = sync.Pool{
		New: func() interface{} {
			return &bytes.Buffer{}
		},
	}
)

func AcquireRecord() *Record {
	return recordPool.Get().(*Record)
}

func ReleaseRecord(r *Record) {
	r.Reset()
	recordPool.Put(r)
}

func acquireBytesBuffer() *bytes.Buffer {
	return bytesBufferPool.Get().(*bytes.Buffer)
}

func releaseBytesBuffer(buf *bytes.Buffer) {
	buf.Reset()
	bytesBufferPool.Put(buf)
}
