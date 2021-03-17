package network

import (
	"fmt"
	"net"
	"net/url"

	"golang.org/x/net/proxy"
)

type socks5Dialer struct {
	proxy.ContextDialer

	bufferSize int
}

func (s socks5Dialer) Dial(protocol, address string) (net.Conn, error) {
	return s.ContextDialer.(proxy.Dialer).Dial(protocol, address)
}

func (s socks5Dialer) TCPBufferSize() int {
	return s.bufferSize
}

func NewSocks5Dialer(baseDialer Dialer, proxyURL *url.URL) (Dialer, error) {
	rv, err := proxy.FromURL(proxyURL, baseDialer)
	if err != nil {
		return nil, fmt.Errorf("cannot initialize socks5 proxy dialer: %w", err)
	}

	return socks5Dialer{
		ContextDialer: rv.(proxy.ContextDialer),
		bufferSize:    baseDialer.TCPBufferSize(),
	}, nil
}
