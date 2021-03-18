package obfuscated2

import (
	"crypto/cipher"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
)

// Connection Type secure. We support only fake tls.
var clientHandshakeMagic = []byte{0xdd, 0xdd, 0xdd, 0xdd}

func ClientHandshake(secret []byte, handshakeFrame *HandhakeFrame) (int16, cipher.Stream, cipher.Stream, error) {
	decHasher := acquireSha256Hasher()
	defer releaseSha256Hasher(decHasher)

	decHasher.Write(handshakeFrame.key()) // nolint: errcheck
	decHasher.Write(secret)               // nolint: errcheck
	decryptor := makeAesCtr(decHasher.Sum(nil), handshakeFrame.iv())

	encHasher := acquireSha256Hasher()
	defer releaseSha256Hasher(encHasher)

	invertedFrame := handshakeFrame.invert()
	encHasher.Write(invertedFrame.key()) // nolint: errcheck
	encHasher.Write(secret)               // nolint: errcheck
	encryptor := makeAesCtr(encHasher.Sum(nil), invertedFrame.iv())

	decryptedFrame := HandhakeFrame{}
	decryptor.XORKeyStream(decryptedFrame.data[:], handshakeFrame.data[:])

	if magic := decryptedFrame.magic(); subtle.ConstantTimeCompare(clientHandshakeMagic, magic) != 1 {
		return 0, nil, nil, fmt.Errorf("unsupported connection type: %s", hex.EncodeToString(magic))
	}

	return decryptedFrame.dc(), encryptor, decryptor, nil
}
