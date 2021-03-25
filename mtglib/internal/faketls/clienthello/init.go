package clienthello

import "errors"

const (
	RandomLen       = 32
	RandomOffset    = 6
	SessionIDOffset = RandomOffset + RandomLen
	MinLen          = SessionIDOffset + 1

	HandshakeTypeClient = 0x01
)

var (
	ErrBadDigest        = errors.New("bad digest")
	ErrAntiReplayAttack = errors.New("antireplay attack was detected")
)
