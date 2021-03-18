package obfuscated2

import (
	"crypto/sha256"
	"hash"
	"sync"
)

var sha256HasherPool = sync.Pool{
	New: func() interface{} {
		return sha256.New()
	},
}

func acquireSha256Hasher() hash.Hash {
	return sha256HasherPool.Get().(hash.Hash)
}

func releaseSha256Hasher(h hash.Hash) {
	h.Reset()
	sha256HasherPool.Put(h)
}
