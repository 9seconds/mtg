package antireplay

import "github.com/9seconds/mtg/v2/mtglib"

type noop struct{}

func (n noop) SeenBefore(_ []byte) bool { return false }

func NewNoop() mtglib.AntiReplayCache {
	return noop{}
}
