package network

import (
	"context"
	"fmt"
	"net/url"

	"github.com/9seconds/mtg/v2/essentials"
	"github.com/9seconds/mtg/v2/mtglib"
	"golang.org/x/net/proxy"
)

type proxyNetwork struct {
	mtglib.Network
	client proxy.ContextDialer
}

func (p proxyNetwork) DialContext(ctx context.Context, network, address string) (essentials.Conn, error) {
	conn, err := p.client.DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}

	return essentials.WrapNetConn(conn), nil
}

func NewProxyNetwork(base mtglib.Network, proxyURL *url.URL) (*proxyNetwork, error) {
	socks, err := proxy.FromURL(proxyURL, base.NativeDialer())
	if err != nil {
		return nil, fmt.Errorf("cannot build proxy dialer: %w", err)
	}

	return &proxyNetwork{
		Network: base,
		client:  socks.(proxy.ContextDialer),
	}, nil
}
