package obfuscation

import (
	"crypto/rand"
	"encoding/binary"
	"slices"
)

// https://core.telegram.org/mtproto/mtproto-transports#transport-obfuscation
const (
	// default DC is nothing is selected
	defaultDC = 2

	// the length of the handshake frame. Always 64 bytes
	hfLen = 64

	hfLenKey            = 32
	hfLenIV             = 16
	hfLenConnectionType = 4

	// A structure of obfuscated handshake frame is following:
	//
	//	[frameOffsetFirst:frameOffsetKey:frameOffsetIV:frameOffsetMagic:frameOffsetDC:frameOffsetEnd].
	//
	//	- 8 bytes of noise
	//	- 32 bytes of AES Key
	//	- 16 bytes of AES IV
	//	- 4 bytes of 'connection type' - this has some setting like a connection type
	//	- 2 bytes of 'DC'. DC is little endian int16
	//	- 2 bytes of noise
	hfOffsetKey            = 8
	hfOffsetIV             = hfOffsetKey + hfLenKey
	hfOffsetConnectionType = hfOffsetIV + hfLenIV
	hfOffsetDC             = hfOffsetConnectionType + hfLenConnectionType
)

// Connection-Type: Secure. We support only fake tls.
var hfConnectionType = [hfLenConnectionType]byte{0xdd, 0xdd, 0xdd, 0xdd}

type handshakeFrame struct {
	data [hfLen]byte
}

func (h *handshakeFrame) key() []byte {
	return h.data[hfOffsetKey : hfOffsetKey+hfLenKey]
}

func (h *handshakeFrame) iv() []byte {
	return h.data[hfOffsetIV : hfOffsetIV+hfLenIV]
}

func (h *handshakeFrame) connectionType() []byte {
	return h.data[hfOffsetConnectionType : hfOffsetConnectionType+hfLenConnectionType]
}

func (h *handshakeFrame) dcSlice() []byte {
	return h.data[hfOffsetDC : hfOffsetDC+2]
}

func (h *handshakeFrame) dc() int {
	idx := int16(binary.LittleEndian.Uint16(h.dcSlice()))

	switch {
	case idx > 0:
		return int(idx)
	case idx < 0:
		return -int(idx)
	}

	return defaultDC
}

func (h *handshakeFrame) revert() {
	slices.Reverse(h.data[hfOffsetKey:hfOffsetConnectionType])
}

func generateHandshake(dc int) handshakeFrame {
	frame := handshakeFrame{}

	for {
		if _, err := rand.Read(frame.data[:]); err != nil {
			panic(err)
		}

		// https://github.com/tdlib/td/blob/master/td/mtproto/TcpTransport.cpp#L157-L158.
		if frame.data[0] == 0xef { // abridged header
			// https://core.telegram.org/mtproto/mtproto-transports#abridged
			continue
		}

		switch binary.LittleEndian.Uint32(frame.data[:4]) {
		case 0x44414548, // HEAD
			0x54534f50, // POST
			0x20544547, // GET
			0x4954504f, // OPTI
			0x02010316, // ????
			0xdddddddd, // PaddedIntermediate header
			0xeeeeeeee: // Intermediate header
			continue
		}

		if frame.data[4]|frame.data[5]|frame.data[6]|frame.data[7] == 0 {
			continue
		}

		copy(frame.connectionType(), hfConnectionType[:])
		binary.LittleEndian.PutUint16(frame.dcSlice(), uint16(dc))

		return frame
	}
}
