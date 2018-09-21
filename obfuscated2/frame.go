package obfuscated2

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"io"

	"github.com/juju/errors"

	"github.com/9seconds/mtg/mtproto"
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

	FrameLen = 64
)

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

// ConnectionType identifies connection type of the handshake frame.
func (f Frame) ConnectionType() (mtproto.ConnectionType, error) {
	return mtproto.ConnectionTagFromHandshake(f.Magic())
}

// Invert inverts frame for extracting encryption keys. Pkease check that link:
// https://blog.susanka.eu/how-telegram-obfuscates-its-mtproto-traffic/
func (f Frame) Invert() Frame {
	reversed := make(Frame, FrameLen)
	copy(reversed, f)

	for i := 0; i < frameLenKey+frameLenIV; i++ {
		reversed[frameOffsetFirst+i] = f[frameOffsetIV-1-i]
	}

	return reversed
}

// ExtractFrame extracts exact obfuscated2 handshake frame from given reader.
func ExtractFrame(conn io.Reader) (Frame, error) {
	frame := make(Frame, FrameLen)
	buf := bytes.NewBuffer(frame)
	buf.Reset()

	if _, err := io.CopyN(buf, conn, FrameLen); err != nil {
		return nil, errors.Annotate(err, "Cannot extract obfuscated header")
	}
	copy(frame, buf.Bytes())

	return frame, nil
}

func generateFrame(connectionType mtproto.ConnectionType) Frame {
	frame := make(Frame, FrameLen)

	for {
		if _, err := rand.Read(frame); err != nil {
			continue
		}
		if frame[0] == 0xef {
			continue
		}

		val := (uint32(frame[3]) << 24) | (uint32(frame[2]) << 16) | (uint32(frame[1]) << 8) | uint32(frame[0])
		if val == 0x44414548 || val == 0x54534f50 || val == 0x20544547 || val == 0x4954504f || val == 0xeeeeeeee {
			continue
		}

		val = (uint32(frame[7]) << 24) | (uint32(frame[6]) << 16) | (uint32(frame[5]) << 8) | uint32(frame[4])
		if val == 0x00000000 {
			continue
		}

		// error has to be checked before calling this function
		tag, _ := connectionType.Tag() // nolint: errcheck, gosec
		copy(frame.Magic(), tag)

		return frame
	}
}
