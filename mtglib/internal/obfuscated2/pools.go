package obfuscated2

import (
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
	handshakeFramePool = sync.Pool{
		New: func() interface{} {
			return &handshakeFrame{}
		},
	}
)

func acquireSha256Hasher() hash.Hash {
	return sha256HasherPool.Get().(hash.Hash)
}

func releaseSha256Hasher(h hash.Hash) {
	h.Reset()
	sha256HasherPool.Put(h)
}

func acquireHandshakeFrame() *handshakeFrame {
	return handshakeFramePool.Get().(*handshakeFrame)
}

func releaseHandshakeFrame(h *handshakeFrame) {
	handshakeFramePool.Put(h)
}
