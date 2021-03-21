package obfuscated2

import "encoding/binary"

const (
	handshakeFrameLen = 64

	handshakeFrameLenKey            = 32
	handshakeFrameLenIV             = 16
	handshakeFrameLenConnectionType = 4
	handshakeFrameLenDC             = 2

	handshakeFrameOffsetStart          = 8
	handshakeFrameOffsetKey            = handshakeFrameOffsetStart
	handshakeFrameOffsetIV             = handshakeFrameOffsetKey + handshakeFrameLenKey
	handshakeFrameOffsetConnectionType = handshakeFrameOffsetIV + handshakeFrameLenIV
	handshakeFrameOffsetDC             = handshakeFrameOffsetConnectionType + handshakeFrameLenConnectionType
	handshakeFrameOffsetEnd            = handshakeFrameOffsetDC + handshakeFrameLenDC
)

// Connection-Type: Secure. We support only fake tls.
var handshakeConnectionType = []byte{0xdd, 0xdd, 0xdd, 0xdd}

// A structure of obfuscated2 handshake frame is following:
//
//    [frameOffsetFirst:frameOffsetKey:frameOffsetIV:frameOffsetMagic:frameOffsetDC:frameOffsetEnd].
//
//    - 8 bytes of noise
//    - 32 bytes of AES Key
//    - 16 bytes of AES IV
//    - 4 bytes of 'connection type' - this has some setting like a connection type
//    - 2 bytes of 'DC'. DC is little endian int16
//    - 2 bytes of noise
type handshakeFrame struct {
	data [handshakeFrameLen]byte
}

func (h *handshakeFrame) dc() int16 {
	data := h.data[handshakeFrameOffsetDC:handshakeFrameOffsetEnd]

	return int16(binary.LittleEndian.Uint16(data))
}

func (h *handshakeFrame) key() []byte {
	return h.data[handshakeFrameOffsetKey:handshakeFrameOffsetIV]
}

func (h *handshakeFrame) iv() []byte {
	return h.data[handshakeFrameOffsetIV:handshakeFrameOffsetConnectionType]
}

func (h *handshakeFrame) connectionType() []byte {
	return h.data[handshakeFrameOffsetConnectionType:handshakeFrameOffsetDC]
}
