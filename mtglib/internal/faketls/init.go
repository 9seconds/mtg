package faketls

import (
	"bytes"
	"errors"
)

const (
	// RandomLen defines a size of the random digest in TLS Hellos.
	RandomLen = 32

	// ClientHelloRandomOffset is an offset in ClientHello record where
	// random digest is started.
	ClientHelloRandomOffset = 6

	// ClientHelloSessionIDOffset is an offset in ClientHello record where
	// SessionID is started.
	ClientHelloSessionIDOffset = ClientHelloRandomOffset + RandomLen

	// ClientHelloMinLen is a minimal possible length of
	// ClientHello record.
	ClientHelloMinLen = 6

	// WelcomePacketRandomOffset is an offset of random in ServerHello
	// packet (including record envelope).
	WelcomePacketRandomOffset = 11

	// HandshakeTypeClient is a value representing a client handshake.
	HandshakeTypeClient = 0x01

	// HandshakeTypeServer is a value representing a server handshake.
	HandshakeTypeServer = 0x02

	// ChangeCipherValue is a value representing a change cipher
	// specification record.
	ChangeCipherValue = 0x01

	// ExtensionSNI is a value for TLS extension 'SNI'.
	ExtensionSNI = 0x00
)

var (
	// ErrBadDigest is returned if given TLS Client Hello mismatches with a
	// derived one.
	ErrBadDigest = errors.New("bad digest")

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
	clientHelloEmptyRandom = bytes.Repeat([]byte{0}, RandomLen)
)
