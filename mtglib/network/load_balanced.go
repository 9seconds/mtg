package network

import (
	"context"
	"math/rand"
	"net"
	"net/url"
)

type loadBalancedDialer struct {
	dialers []Dialer
}

func (l loadBalancedDialer) Dial(network, address string) (net.Conn, error) {
	return l.DialContext(context.Background(), network, address)
}

func (l loadBalancedDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
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

func NewLoadBalancedDialer(baseDialer Dialer, proxyURLs []*url.URL) (Dialer, error) {
	switch len(proxyURLs) {
	case 0:
		return baseDialer, nil
	case 1:
		return NewSocks5Dialer(baseDialer, proxyURLs[0])
	}

	dialers := []Dialer{}

	for _, u := range proxyURLs {
		dialer, err := NewSocks5Dialer(newProxyDialer(baseDialer, u), u)
		if err != nil {
			return nil, err
		}

		dialers = append(dialers, dialer)
	}

	return loadBalancedDialer{
		dialers: dialers,
	}, nil
}
