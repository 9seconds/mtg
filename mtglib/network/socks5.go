package network

import (
	"fmt"
	"net/url"
	"time"

	"golang.org/x/net/proxy"
)

func NewSocks5Dialer(proxyUrl *url.URL, timeout time.Duration, bufferSize int) (Dialer, error) {
	dialer, err := NewDefaultDialer(timeout, bufferSize)
	if err != nil {
		return nil, fmt.Errorf("cannot initialize base dialer: %w", err)
	}

	rv, err := proxy.FromURL(proxyUrl, dialer.(*defaultDialer))
	if err != nil {
		return nil, fmt.Errorf("cannot initialize socks5 proxy dialer: %w", err)
	}

	return rv.(Dialer), nil
}
