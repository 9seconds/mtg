package ipblocklist

import (
	"net"

	"github.com/9seconds/mtg/v2/mtglib"
)

type noop struct{}

func (n noop) Contains(ip net.IP) bool { return false }
func (n noop) Shutdown()               {}

func NewNoop() mtglib.IPBlocklist {
	return noop{}
}
