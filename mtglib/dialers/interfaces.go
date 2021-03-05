package dialers

import (
	"context"
	"net"
)

type BaseDialer interface {
	Dial(network, address string) (net.Conn, error)
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}
