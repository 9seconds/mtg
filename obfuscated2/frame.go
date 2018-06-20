package obfuscated2

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"io"

	"github.com/juju/errors"
)

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

// Frame represents handshake frame. Telegram sends 64 bytes of obfuscated2
// initialization data first.
// https://blog.susanka.eu/how-telegram-obfuscates-its-mtproto-traffic/
type Frame []byte

// Key returns AES encryption key.
func (f Frame) Key() []byte {
	return f[frameOffsetFirst:frameOffsetKey]
}

// IV returns AES encryption initialization vector
func (f Frame) IV() []byte {
	return f[frameOffsetKey:frameOffsetIV]
}

// Magic returns magic bytes from last 8 bytes of frame. Telegram checks
// for values there. If after decryption magic is not as expected,
// connection considered as failed.
func (f Frame) Magic() []byte {
	return f[frameOffsetIV:frameOffsetMagic]
}

// DC returns number of datacenter IP client wants to use.
func (f Frame) DC() (n int16) {
	buf := bytes.NewReader(f[frameOffsetMagic:frameOffsetDC])
	if err := binary.Read(buf, binary.LittleEndian, &n); err != nil {
		n = 1
	}

	return
}

// Valid checks that *decrypted* frame is valid. Only magic bytes are checked.
func (f Frame) Valid() bool {
	return bytes.Equal(f.Magic(), tgMagicBytes)
}

// Invert inverts frame for extracting encryption keys. Pkease check that link:
// https://blog.susanka.eu/how-telegram-obfuscates-its-mtproto-traffic/
func (f Frame) Invert() *Frame {
	reversed := MakeFrame()
	copy(*reversed, f)

	for i := 0; i < frameLenKey+frameLenIV; i++ {
		(*reversed)[frameOffsetFirst+i] = f[frameOffsetIV-1-i]
	}

	return reversed
}

// ExtractFrame extracts exact obfuscated2 handshake frame from given reader.
func ExtractFrame(conn io.Reader) (*Frame, error) {
	frame := MakeFrame()
	buf := bytes.NewBuffer(*frame)
	buf.Reset()

	if _, err := io.CopyN(buf, conn, FrameLen); err != nil {
		ReturnFrame(frame)
		return nil, errors.Annotate(err, "Cannot extract obfuscated header")
	}
	copy(*frame, buf.Bytes())

	return frame, nil
}

func generateFrame() *Frame {
	frame := MakeFrame()
	data := *frame

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

		return frame
	}
}
