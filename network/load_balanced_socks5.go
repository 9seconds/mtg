package network

import (
	"context"
	"fmt"
	"math/rand"
	"net/url"

	"github.com/9seconds/mtg/v2/essentials"
)

type loadBalancedSocks5Dialer struct {
	dialers []Dialer
}

func (l loadBalancedSocks5Dialer) Dial(network, address string) (essentials.Conn, error) {
	return l.DialContext(context.Background(), network, address)
}

func (l loadBalancedSocks5Dialer) DialContext(ctx context.Context, network, address string) (essentials.Conn, error) {
	length := len(l.dialers)
	start := rand.Intn(length)
	moved := false

	for i := start; i != start || !moved; i = (i + 1) % length {
		moved = true

		if conn, err := l.dialers[i].DialContext(ctx, network, address); err == nil {
			return conn, nil
		}
	}

	return nil, ErrCannotDialWithAllProxies
}

// NewLoadBalancedSocks5Dialer builds a new load balancing SOCKS5 dialer.
//
// The main difference from one which is made by NewSocks5Dialer is that we
// actually have a list of these proxies. When dial is requested, any proxy is
// picked and used. If proxy fails for some reason, we try another one.
//
// So, it is mostly useful if you have some routes with proxies which are not
// always online or having buggy network.
func NewLoadBalancedSocks5Dialer(baseDialer Dialer, proxyURLs []*url.URL) (Dialer, error) {
	dialers := make([]Dialer, 0, len(proxyURLs))

	for _, u := range proxyURLs {
		dialer, err := NewSocks5Dialer(newProxyDialer(baseDialer, u), u)
		if err != nil {
			return nil, fmt.Errorf("cannot build dialer for %s: %w", u.String(), err)
		}

		dialers = append(dialers, dialer)
	}

	return loadBalancedSocks5Dialer{
		dialers: dialers,
	}, nil
}
