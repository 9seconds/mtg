package obfuscated2

import (
	"crypto/cipher"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"io"
)

// Connection Type secure. We support only fake tls.
var clientHandshakeConnectionType = []byte{0xdd, 0xdd, 0xdd, 0xdd}

func ClientHandshake(secret []byte, reader io.Reader) (int16, cipher.Stream, cipher.Stream, error) {
	handshakeFrame := acquireHandshakeFrame()
	defer releaseHandshakeFrame(handshakeFrame)

	if _, err := io.ReadFull(reader, handshakeFrame.data[:]); err != nil {
		return 0, nil, nil, fmt.Errorf("cannot read frame: %w", err)
	}

	decHasher := acquireSha256Hasher()
	defer releaseSha256Hasher(decHasher)

	decHasher.Write(handshakeFrame.key()) // nolint: errcheck
	decHasher.Write(secret)               // nolint: errcheck
	decryptor := makeAesCtr(decHasher.Sum(nil), handshakeFrame.iv())

	encHasher := acquireSha256Hasher()
	defer releaseSha256Hasher(encHasher)

	invertedFrame := acquireHandshakeFrame()
	defer releaseHandshakeFrame(invertedFrame)

	for i, v := range handshakeFrame.data {
		invertedFrame.data[handshakeFrameLen-1-i] = v
	}

	encHasher.Write(invertedFrame.key()) // nolint: errcheck
	encHasher.Write(secret)              // nolint: errcheck
	encryptor := makeAesCtr(encHasher.Sum(nil), invertedFrame.iv())

	decryptor.XORKeyStream(handshakeFrame.data[:], handshakeFrame.data[:])

	if val := handshakeFrame.connectionType(); subtle.ConstantTimeCompare(clientHandshakeConnectionType, val) != 1 {
		return 0, nil, nil, fmt.Errorf("unsupported connection type: %s", hex.EncodeToString(val))
	}

	return handshakeFrame.dc(), encryptor, decryptor, nil
}
