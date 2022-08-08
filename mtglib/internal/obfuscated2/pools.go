package obfuscated2

import (
	"bytes"
	"crypto/sha256"
	"hash"
	"sync"
)

var (
	sha256HasherPool = sync.Pool{
		New: func() interface{} {
			return sha256.New()
		},
	}
	bytesBufferPool = sync.Pool{
		New: func() interface{} {
			return &bytes.Buffer{}
		},
	}
)

func acquireSha256Hasher() hash.Hash {
	return sha256HasherPool.Get().(hash.Hash) //nolint: forcetypeassert
}

func releaseSha256Hasher(h hash.Hash) {
	h.Reset()
	sha256HasherPool.Put(h)
}

func acquireBytesBuffer() *bytes.Buffer {
	return bytesBufferPool.Get().(*bytes.Buffer) //nolint: forcetypeassert
}

func releaseBytesBuffer(buf *bytes.Buffer) {
	buf.Reset()
	bytesBufferPool.Put(buf)
}
