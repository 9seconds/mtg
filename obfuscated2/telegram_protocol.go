package obfuscated2

import (
	"crypto/rand"
	"fmt"

	"github.com/9seconds/mtg/conntypes"
	"github.com/9seconds/mtg/protocol"
	"github.com/9seconds/mtg/telegram"
	"github.com/9seconds/mtg/utils"
	"github.com/9seconds/mtg/wrappers/stream"
)

func TelegramProtocol(req *protocol.TelegramRequest) (conntypes.StreamReadWriteCloser, error) {
	conn, err := telegram.Direct.Dial(req.ClientProtocol.DC(),
		req.ClientProtocol.ConnectionProtocol())
	if err != nil {
		return nil, fmt.Errorf("cannot dial to telegram: %w", err)
	}

	conn = stream.NewTimeout(conn)
	conn = stream.NewCtx(req.Ctx, req.Cancel, conn)
	fm := generateFrame(req.ClientProtocol)
	data := fm.Bytes()

	encryptor := utils.MakeStreamCipher(fm.Key(), fm.IV())
	decryptedFrame := fm.Invert()
	decryptor := utils.MakeStreamCipher(decryptedFrame.Key(), decryptedFrame.IV())

	copyFrame := make([]byte, frameLen)
	copy(copyFrame[:frameOffsetIV], data[:frameOffsetIV])
	encryptor.XORKeyStream(data, data)
	copy(data[:frameOffsetIV], copyFrame[:frameOffsetIV])

	if _, err := conn.Write(data); err != nil {
		return nil, fmt.Errorf("cannot write handshake frame to telegram: %w", err)
	}

	return stream.NewObfuscated2(conn, encryptor, decryptor), nil
}

func generateFrame(cp protocol.ClientProtocol) (fm Frame) {
	data := fm.Bytes()

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

		copy(fm.Magic(), cp.ConnectionType().Tag())

		return
	}
}
