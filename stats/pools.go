package stats

import "sync"

var streamInfoPool = sync.Pool{
	New: func() interface{} {
		return &streamInfo{
			tags: make(map[string]string),
		}
	},
}

func acquireStreamInfo() *streamInfo {
	return streamInfoPool.Get().(*streamInfo) //nolint: forcetypeassert
}

func releaseStreamInfo(info *streamInfo) {
	info.Reset()
	streamInfoPool.Put(info)
}
