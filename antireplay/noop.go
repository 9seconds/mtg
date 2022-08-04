package antireplay

import "github.com/9seconds/mtg/v2/mtglib"

type noop struct{}

func (n noop) SeenBefore(_ []byte) bool { return false }

// NewNoop returns an implementation that does nothing. A corresponding method
// always returns false, so this cache accepts everything you pass to it.
func NewNoop() mtglib.AntiReplayCache {
	return noop{}
}
