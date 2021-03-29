package stats

import "sync"

var streamInfoPool = sync.Pool{
	New: func() interface{} {
		return streamInfo{}
	},
}

func acquireStreamInfo() streamInfo {
	return streamInfoPool.Get().(streamInfo)
}

func releaseStreamInfo(info streamInfo) {
	info.Reset()
	streamInfoPool.Put(info)
}
