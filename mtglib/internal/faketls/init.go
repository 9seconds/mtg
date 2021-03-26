package faketls

import "errors"

const (
	RandomLen = 32

	ClientHelloRandomOffset    = 6
	ClientHelloSessionIDOffset = ClientHelloRandomOffset + RandomLen
	ClientHelloMinLen          = 4

	WelcomePacketRandomOffset = 11

	HandshakeTypeClient = 0x01
	HandshakeTypeServer = 0x02

	ChangeCipherValue = 0x01

	ExtensionSNI = 0x00
)

var (
	ErrBadDigest        = errors.New("bad digest")
	ErrAntiReplayAttack = errors.New("antireplay attack was detected")

	serverHelloSuffix = []byte{
		0x00,       // no compression
		0x00, 0x2e, // 46 bytes of data
		0x00, 0x2b, // Extension - Supported Versions
		0x00, 0x02, // 2 bytes are following
		0x03, 0x04, // TLS 1.3
		0x00, 0x33, // Extension - Key Share
		0x00, 0x24, // 36 bytes
		0x00, 0x1d, // x25519 curve
		0x00, 0x20, // 32 bytes of key
	}
)
