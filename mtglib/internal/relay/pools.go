package relay

import (
	"bufio"
	"io"
	"net"
	"sync"
)

var syncPairPool = sync.Pool{
	New: func() interface{} {
		return &syncPair{
			writer:  bufio.NewWriterSize(nil, writerBufferSize),
			copyBuf: make([]byte, copyBufferSize),
		}
	},
}

func acquireSyncPair(reader net.Conn, writer io.Writer) *syncPair {
	sp := syncPairPool.Get().(*syncPair) // nolint: forcetypeassert
	sp.writer.Reset(writer)
	sp.reader = reader

	return sp
}

func releaseSyncPair(sp *syncPair) {
	sp.writer.Reset(nil)
	sp.reader = nil
	syncPairPool.Put(sp)
}
