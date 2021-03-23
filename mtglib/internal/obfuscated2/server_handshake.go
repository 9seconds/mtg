package obfuscated2

import (
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"net"
)

type serverHandshakeFrame struct {
	handshakeFrame
}

func (s *serverHandshakeFrame) decryptor() cipher.Stream {
	return makeAesCtr(s.key(), s.iv())
}

func (s *serverHandshakeFrame) encryptor() cipher.Stream {
	arr := serverHandshakeFrame{}
	invertByteSlices(arr.data[:], s.data[:])

	return makeAesCtr(arr.key(), arr.iv())
}

func ServerHandshake(conn net.Conn) (cipher.Stream, cipher.Stream, error) {
	handshake := generateServerHanshakeFrame()
	copyHandshake := handshake
	encryptor := handshake.encryptor()
	decryptor := handshake.decryptor()

	encryptor.XORKeyStream(handshake.data[:], handshake.data[:])
	copy(handshake.key(), copyHandshake.key())
	copy(handshake.iv(), copyHandshake.iv())

	if _, err := conn.Write(handshake.data[:]); err != nil {
		return nil, nil, fmt.Errorf("cannot send a handshake frame to telegram: %w", err)
	}

	return encryptor, decryptor, nil
}

func generateServerHanshakeFrame() serverHandshakeFrame {
	frame := serverHandshakeFrame{}

	for {
		if _, err := rand.Read(frame.data[:]); err != nil {
			panic(err)
		}

		if frame.data[0] == 0xef {
			continue
		}

		switch binary.LittleEndian.Uint32(frame.data[:4]) {
		case 0x44414548, 0x54534f50, 0x20544547, 0x4954504f, 0xeeeeeeee:
			continue
		}

		if (frame.data[4] | frame.data[5] | frame.data[6] | frame.data[7]) == 0 {
			continue
		}

		copy(frame.connectionType(), handshakeConnectionType)

		return frame
	}
}
