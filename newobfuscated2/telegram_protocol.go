package newobfuscated2

import (
	"crypto/rand"

	"github.com/juju/errors"

	"github.com/9seconds/mtg/newprotocol"
	"github.com/9seconds/mtg/newwrappers"
)

type TelegramProtocol struct {
	newprotocol.BaseProtocol
}

func (t *TelegramProtocol) Handshake(socketRaw newwrappers.Wrap, client *ClientProtocol) (newwrappers.StreamReadWriteCloser, error) {
	socket := socketRaw.(newwrappers.StreamReadWriteCloser)
	fm := generateFrame(client)
	data := fm.bytes()

	encryptor := makeStreamCipher(fm.key(), fm.iv())
	decryptedFrame := fm.invert()
	decryptor := makeStreamCipher(decryptedFrame.key(), decryptedFrame.iv())

	copyFrame := make([]byte, frameLen)
	copy(copyFrame[:frameOffsetIV], data[:frameOffsetIV])
	encryptor.XORKeyStream(data, data)
	copy(data[:frameOffsetIV], copyFrame[:frameOffsetIV])

	if _, err := socket.Write(data); err != nil {
		return nil, errors.Annotate(err, "Cannot write handshate frame to Telegram")
	}

	return newwrappers.NewObfuscated2(socket, encryptor, decryptor), nil
}

func generateFrame(client *ClientProtocol) (fm frame) {
	for {
		data := fm.bytes()
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

		copy(fm.magic(), client.ConnectionType.Tag())

		return
	}
}
