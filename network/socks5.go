package network

import (
	"fmt"
	"net/url"

	"golang.org/x/net/proxy"
)

func NewSocks5Dialer(baseDialer Dialer, proxyURL *url.URL) (Dialer, error) {
	rv, err := proxy.FromURL(proxyURL, baseDialer)
	if err != nil {
		return nil, fmt.Errorf("cannot initialize socks5 proxy dialer: %w", err)
	}

	return rv.(Dialer), nil
}
