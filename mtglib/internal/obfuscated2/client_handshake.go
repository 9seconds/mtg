package obfuscated2

import (
	"crypto/cipher"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"io"
)

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

func ClientHandshake(secret []byte, reader io.Reader) (int, cipher.Stream, cipher.Stream, error) {
	handshake := clientHandhakeFrame{}

	if _, err := io.ReadFull(reader, handshake.data[:]); err != nil {
		return 0, nil, nil, fmt.Errorf("cannot read frame: %w", err)
	}

	decryptor := handshake.decryptor(secret)
	encryptor := handshake.encryptor(secret)

	decryptor.XORKeyStream(handshake.data[:], handshake.data[:])

	if val := handshake.connectionType(); subtle.ConstantTimeCompare(handshakeConnectionType, val) != 1 {
		return 0, nil, nil, fmt.Errorf("unsupported connection type: %s", hex.EncodeToString(val))
	}

	return handshake.dc(), encryptor, decryptor, nil
}
