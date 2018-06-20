package obfuscated2

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFrameKey(t *testing.T) {
	toCompare := make([]byte, 32)
	for i := 0; i < 32; i++ {
		toCompare[i] = byte(1)
	}

	assert.Equal(t, toCompare, makeFrame().Key())
}

func TestFrameIV(t *testing.T) {
	toCompare := make([]byte, 16)
	for i := 0; i < 16; i++ {
		toCompare[i] = byte(2)
	}

	assert.Equal(t, toCompare, makeFrame().IV())
}

func TestFrameMagic(t *testing.T) {
	toCompare := make([]byte, 4)
	for i := 0; i < 4; i++ {
		toCompare[i] = tgMagicByte
	}

	assert.Equal(t, toCompare, makeFrame().Magic())
}

func TestFrameDC(t *testing.T) {
	assert.Equal(t, int16(771), makeFrame().DC())
}

func TestFrameValid(t *testing.T) {
	frame := makeFrame()
	assert.True(t, frame.Valid())

	frame[8+32+16+2] = byte(3)
	assert.False(t, frame.Valid())
}

func TestFrameDoubleInvert(t *testing.T) {
	frame := makeFrame()
	assert.True(t, bytes.Equal(frame, *frame.Invert().Invert()))
}

func TestFrameInvert(t *testing.T) {
	frame := makeFrame()
	reversed := frame.Invert()

	assert.Exactly(t, frame[:8], (*reversed)[:8])
	assert.Exactly(t, frame[56:], (*reversed)[56:])

	toCompare := make([]byte, 48)
	for i := 0; i < 48; i++ {
		toCompare[i] = frame[55-i]
	}
	assert.Equal(t, []byte((*reversed)[8:56]), toCompare)
}

func TestFrameGenerateValid(t *testing.T) {
	assert.True(t, generateFrame().Valid())
}

func makeFrame() Frame {
	f := make(Frame, FrameLen)

	for i := 8; i < (8 + 32); i++ {
		f[i] = byte(1)
	}
	for i := (8 + 32); i < (8 + 32 + 16); i++ {
		f[i] = byte(2)
	}
	for i := (8 + 32 + 16); i < (8 + 32 + 16 + 4); i++ {
		f[i] = tgMagicByte
	}
	for i := (8 + 32 + 16 + 4); i < (8 + 32 + 16 + 4 + 2); i++ {
		f[i] = byte(3)
	}

	return f
}
