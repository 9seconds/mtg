package obfuscated2

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"io"
)

// Connection Type secure. We support only fake tls.
var clientHandshakeConnectionType = []byte{0xdd, 0xdd, 0xdd, 0xdd}

func ClientHandshake(secret []byte, reader io.Reader) (int16, cipher.Stream, cipher.Stream, error) {
	handshake := handshakeFrame{}

	if _, err := io.ReadFull(reader, handshake.data[:]); err != nil {
		return 0, nil, nil, fmt.Errorf("cannot read frame: %w", err)
	}

	decHasher := acquireSha256Hasher()
	defer releaseSha256Hasher(decHasher)

	decHasher.Write(handshake.key()) // nolint: errcheck
	decHasher.Write(secret)          // nolint: errcheck
	decryptor := makeAesCtr(decHasher.Sum(nil), handshake.iv())

	encHasher := acquireSha256Hasher()
	defer releaseSha256Hasher(encHasher)

	invertedHandshake := handshakeFrame{}

	for i, v := range handshake.data {
		invertedHandshake.data[handshakeFrameLen-1-i] = v
	}

	encHasher.Write(invertedHandshake.key()) // nolint: errcheck
	encHasher.Write(secret)                  // nolint: errcheck
	encryptor := makeAesCtr(encHasher.Sum(nil), invertedHandshake.iv())

	decryptor.XORKeyStream(handshake.data[:], handshake.data[:])

	if val := handshake.connectionType(); subtle.ConstantTimeCompare(clientHandshakeConnectionType, val) != 1 {
		return 0, nil, nil, fmt.Errorf("unsupported connection type: %s", hex.EncodeToString(val))
	}

	return handshake.dc(), encryptor, decryptor, nil
}

func makeAesCtr(key, iv []byte) cipher.Stream {
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}

	return cipher.NewCTR(block, iv)
}
