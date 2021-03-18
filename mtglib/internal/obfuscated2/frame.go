package obfuscated2

import (
	"encoding/binary"
	"fmt"
	"io"
)

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
type HandhakeFrame struct {
	data [handshakeFrameLen]byte
}

func (f *HandhakeFrame) Fingerprint() []byte {
	return f.data[handshakeFrameOffsetStart:handshakeFrameOffsetEnd]
}

func (f *HandhakeFrame) dc() int16 {
	data := f.data[handshakeFrameOffsetDC:handshakeFrameOffsetEnd]

	return int16(binary.LittleEndian.Uint16(data))
}

func (f *HandhakeFrame) key() []byte {
	return f.data[handshakeFrameLenKey:handshakeFrameOffsetIV]
}

func (f *HandhakeFrame) iv() []byte {
	return f.data[handshakeFrameOffsetIV:handshakeFrameOffsetMagic]
}

func (f *HandhakeFrame) magic() []byte {
	return f.data[handshakeFrameOffsetMagic:handshakeFrameOffsetDC]
}

func (f *HandhakeFrame) invert() *HandhakeFrame {
	newFrame := &HandhakeFrame{}

	for i, v := range f.data {
		newFrame.data[handshakeFrameLen-1-i] = v
	}

	return newFrame
}

func ReadHandshakeFrame(reader io.Reader) (*HandhakeFrame, error) {
	frame := &HandhakeFrame{}

	if _, err := io.ReadFull(reader, frame.data[:]); err != nil {
		return nil, fmt.Errorf("cannot read frame data: %w", err)
	}

	return frame, nil
}
