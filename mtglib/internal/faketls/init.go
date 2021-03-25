package faketls

import "errors"

const (
	RandomLen = 32

	ClientHelloRandomOffset    = 6
	ClientHelloSessionIDOffset = ClientHelloRandomOffset + RandomLen
	ClientHelloMinLen      = ClientHelloSessionIDOffset + 1

	HandshakeTypeClient = 0x01
)

var (
	ErrBadDigest        = errors.New("bad digest")
	ErrAntiReplayAttack = errors.New("antireplay attack was detected")
)
