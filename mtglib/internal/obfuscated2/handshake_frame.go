package obfuscated2

import "encoding/binary"

const (
	handshakeFrameLen = 64

	handshakeFrameLenKey   = 32
	handshakeFrameLenIV    = 16
	handshakeFrameLenMagic = 4
	handshakeFrameLenDC    = 2

	handshakeFrameOffsetStart = 8
	handshakeFrameOffsetKey   = handshakeFrameOffsetStart
	handshakeFrameOffsetIV    = handshakeFrameOffsetKey + handshakeFrameLenKey
	handshakeFrameOffsetMagic = handshakeFrameOffsetIV + handshakeFrameLenIV
	handshakeFrameOffsetDC    = handshakeFrameOffsetMagic + handshakeFrameLenMagic
	handshakeFrameOffsetEnd   = handshakeFrameOffsetDC + handshakeFrameLenDC
)

// A structure of obfuscated2 handshake frame is following:
//
//    [frameOffsetFirst:frameOffsetKey:frameOffsetIV:frameOffsetMagic:frameOffsetDC:frameOffsetEnd].
//
//    - 8 bytes of noise
//    - 32 bytes of AES Key
//    - 16 bytes of AES IV
//    - 4 bytes of 'magic' - this has some settings like a connection type
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
	return h.data[handshakeFrameLenKey:handshakeFrameOffsetIV]
}

func (h *handshakeFrame) iv() []byte {
	return h.data[handshakeFrameOffsetIV:handshakeFrameOffsetMagic]
}

func (h *handshakeFrame) magic() []byte {
	return h.data[handshakeFrameOffsetMagic:handshakeFrameOffsetDC]
}
