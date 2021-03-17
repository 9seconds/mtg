package network

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"net/url"
)

type loadBalancedSocks5Dialer struct {
	dialers    []Dialer
	bufferSize int
}

func (l loadBalancedSocks5Dialer) Dial(network, address string) (net.Conn, error) {
	return l.DialContext(context.Background(), network, address)
}

func (l loadBalancedSocks5Dialer) TCPBufferSize() int {
	return l.bufferSize
}

func (l loadBalancedSocks5Dialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
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
		dialers:    dialers,
		bufferSize: baseDialer.TCPBufferSize(),
	}, nil
}
