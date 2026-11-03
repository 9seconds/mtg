package network

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/9seconds/mtg/v2/essentials"
	"github.com/9seconds/mtg/v2/mtglib"
	"golang.org/x/net/proxy"
)

type proxyNetwork struct {
	mtglib.Network
	client proxy.ContextDialer
}

func (p proxyNetwork) Dial(network, address string) (essentials.Conn, error) {
	return p.DialContext(context.Background(), network, address)
}

func (p proxyNetwork) DialContext(ctx context.Context, network, address string) (essentials.Conn, error) {
	conn, err := p.client.DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}

	return essentials.WrapNetConn(conn), nil
}

func (p proxyNetwork) MakeHTTPClient(
	dialFunc func(context.Context, string, string) (essentials.Conn, error),
) *http.Client {
	if dialFunc == nil {
		dialFunc = p.DialContext
	}

	return p.Network.MakeHTTPClient(dialFunc)
}

// proxyServerDialer is the dialer used to connect to a SOCKS upstream. It
// copies timeout and fallback-delay from the base dialer but drops the custom
// Resolver: the user-supplied DoT/DoH resolver is for bypassing DPI on public
// names (Telegram DCs, fronting host), whereas SOCKS upstream addresses are
// usually internal (docker compose, k8s, /etc/hosts) and must be resolved by
// the system resolver. See https://github.com/9seconds/mtg/issues/439.
func proxyServerDialer(base mtglib.Network) *net.Dialer {
	nd := base.NativeDialer()

	return &net.Dialer{
		Timeout:       nd.Timeout,
		FallbackDelay: nd.FallbackDelay,
	}
}

func NewProxyNetwork(base mtglib.Network, proxyURL *url.URL) (*proxyNetwork, error) {
	socks, err := proxy.FromURL(proxyURL, proxyServerDialer(base))
	if err != nil {
		return nil, fmt.Errorf("cannot build proxy dialer: %w", err)
	}

	return &proxyNetwork{
		Network: base,
		client:  socks.(proxy.ContextDialer),
	}, nil
}
