package ipblocklist

import (
	"net"
	"time"

	"github.com/9seconds/mtg/v2/mtglib"
)

type noop struct{}

func (n noop) Contains(ip net.IP) bool      { return false }
func (n noop) Run(updateEach time.Duration) {}
func (n noop) Shutdown()                    {}

// NewNoop returns a dummy ipblocklist which allows all incoming connections.
func NewNoop() mtglib.IPBlocklist {
	return noop{}
}
