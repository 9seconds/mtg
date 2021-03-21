package obfuscated2

import (
	"crypto/cipher"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"io"
)

func ClientHandshake(secret []byte, reader io.Reader) (int16, cipher.Stream, cipher.Stream, error) {
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
