package obfuscated2

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"io"

	"github.com/juju/errors"
)

// https://blog.susanka.eu/how-telegram-obfuscates-its-mtproto-traffic/
// [frameOffsetFirst:frameOffsetKey:frameOffsetIV:frameOffsetMagic:frameOffsetDC:frameOffsetEnd]
const (
	frameLenKey   = 32
	frameLenIV    = 16
	frameLenMagic = 4
	frameLenDC    = 2

	frameOffsetFirst = 8
	frameOffsetKey   = frameOffsetFirst + frameLenKey
	frameOffsetIV    = frameOffsetKey + frameLenIV
	frameOffsetMagic = frameOffsetIV + frameLenMagic
	frameOffsetDC    = frameOffsetMagic + frameLenDC

	tgMagicByte = byte(239)

	FrameLen = 64
)

var tgMagicBytes = []byte{tgMagicByte, tgMagicByte, tgMagicByte, tgMagicByte}

type Frame []byte

func (f Frame) Key() []byte {
	return f[frameOffsetFirst:frameOffsetKey]
}

func (f Frame) IV() []byte {
	return f[frameOffsetKey:frameOffsetIV]
}

func (f Frame) Magic() []byte {
	return f[frameOffsetIV:frameOffsetMagic]
}

func (f Frame) DC() (n int16) {
	buf := bytes.NewReader(f[frameOffsetMagic:frameOffsetDC])
	binary.Read(buf, binary.LittleEndian, &n)

	if n < 0 {
		n = -n
	} else if n == 0 {
		n = 1
	}

	return n - 1
}

func (f Frame) Valid() bool {
	return bytes.Equal(f.Magic(), tgMagicBytes)
}

func (f Frame) Invert() Frame {
	reversed := make(Frame, FrameLen)
	copy(reversed, f)

	for i := 0; i < frameLenKey+frameLenIV; i++ {
		reversed[frameOffsetFirst+i] = f[frameOffsetIV-1-i]
	}

	return reversed
}

func ExtractFrame(conn io.Reader) (Frame, error) {
	buf := &bytes.Buffer{}
	if _, err := io.CopyN(buf, conn, FrameLen); err != nil {
		return nil, errors.Annotate(err, "Cannot extract obfuscated header")
	}

	return Frame(buf.Bytes()), nil
}

func generateFrame() Frame {
	data := make(Frame, FrameLen)

	for {
		if _, err := rand.Read(data); err != nil {
			continue
		}
		if data[0] == 0xef {
			continue
		}

		val := (uint32(data[3]) << 24) | (uint32(data[2]) << 16) | (uint32(data[1]) << 8) | uint32(data[0])
		if val == 0x44414548 || val == 0x54534f50 || val == 0x20544547 || val == 0x4954504f || val == 0xeeeeeeee {
			continue
		}

		val = (uint32(data[7]) << 24) | (uint32(data[6]) << 16) | (uint32(data[5]) << 8) | uint32(data[4])
		if val == 0x00000000 {
			continue
		}

		copy(data.Magic(), tgMagicBytes)

		return data
	}
}
