package record

import (
	"sync"
)

var recordPool = sync.Pool{
	New: func() any {
		return &Record{}
	},
}

func AcquireRecord() *Record {
	return recordPool.Get().(*Record) //nolint: forcetypeassert
}

func ReleaseRecord(r *Record) {
	r.Reset()
	recordPool.Put(r)
}
