package obfuscated2

import "crypto/cipher"

type clientHandhakeFrame struct {
	handshakeFrame
}

func (c *clientHandhakeFrame) decryptor(secret []byte) cipher.Stream {
	hasher := acquireSha256Hasher()
	defer releaseSha256Hasher(hasher)

	hasher.Write(c.key()) // nolint: errcheck
	hasher.Write(secret)  // nolint: errcheck

	return makeAesCtr(hasher.Sum(nil), c.iv())
}

func (c *clientHandhakeFrame) encryptor(secret []byte) cipher.Stream {
	arr := clientHandhakeFrame{}
	invertByteSlices(arr.data[:], c.data[:])

	hasher := acquireSha256Hasher()
	defer releaseSha256Hasher(hasher)

	hasher.Write(arr.key()) // nolint: errcheck
	hasher.Write(secret)    // nolint: errcheck

	return makeAesCtr(hasher.Sum(nil), arr.iv())
}
