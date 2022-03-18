package obfuscated2

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
)

func FuzzServerGenerateHandshakeFrame(f *testing.F) {
	f.Fuzz(func(t *testing.T, arg int) {
		frame := generateServerHanshakeFrame()

		assert.NotEqualValues(t, 0xef, frame.data[0])

		firstBytes := binary.LittleEndian.Uint32(frame.data[:4])
		assert.NotEqualValues(t, 0x44414548, firstBytes)
		assert.NotEqualValues(t, 0x54534f50, firstBytes)
		assert.NotEqualValues(t, 0x20544547, firstBytes)
		assert.NotEqualValues(t, 0x4954504f, firstBytes)
		assert.NotEqualValues(t, 0xeeeeeeee, firstBytes)

		assert.NotEqualValues(
			t,
			0,
			frame.data[4]|frame.data[5]|frame.data[6]|frame.data[7])

		assert.Equal(t, handshakeConnectionType, frame.connectionType())
	})
}
