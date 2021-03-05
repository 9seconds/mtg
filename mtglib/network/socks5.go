package network

import (
	"fmt"
	"net/url"

	"golang.org/x/net/proxy"
)

func NewSocks5Dialer(proxyURL *url.URL, base Dialer) (Dialer, error) {
	rv, err := proxy.FromURL(proxyURL, base)
	if err != nil {
		return nil, fmt.Errorf("cannot initialize socks5 proxy dialer: %w", err)
	}

	return rv.(Dialer), nil
}
