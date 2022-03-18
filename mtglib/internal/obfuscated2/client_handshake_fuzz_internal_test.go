package obfuscated2

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

var FuzzClientHandshakeSecret = []byte{1, 2, 3}

func FuzzClientHandshake(f *testing.F) {
	f.Add([]byte{1, 2, 3})

	f.Fuzz(func(t *testing.T, frame []byte) {
		data := bytes.NewReader(frame)

		if _, _, _, err := ClientHandshake(FuzzClientHandshakeSecret, data); err != nil {
			return
		}

		handshake := clientHandhakeFrame{}
		require.Len(t, frame, handshakeFrameLen)

		copy(handshake.data[:], frame)

		decryptor := handshake.decryptor(FuzzClientHandshakeSecret)
		decryptor.XORKeyStream(handshake.data[:], handshake.data[:])

		require.Equal(t, handshakeConnectionType, handshake.connectionType())
	})
}
