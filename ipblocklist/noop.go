package ipblocklist

import (
	"net"

	"github.com/9seconds/mtg/v2/mtglib"
)

type noop struct{}

func (n noop) Contains(ip net.IP) bool { return false }

// NewNoop returns a dummy ipblocklist which allows all incoming
// connections.
func NewNoop() mtglib.IPBlocklist {
	return noop{}
}
