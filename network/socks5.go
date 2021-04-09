package network

import (
	"fmt"
	"net/url"

	"golang.org/x/net/proxy"
)

// NewSocks5Dialer build a new dialer from a given one (so, in theory
// you can chain here). Proxy parameters are passed with URI in a form of:
//
//     socks5://[user:[password]]@host:port
func NewSocks5Dialer(baseDialer Dialer, proxyURL *url.URL) (Dialer, error) {
	rv, err := proxy.FromURL(proxyURL, baseDialer)
	if err != nil {
		return nil, fmt.Errorf("cannot initialize socks5 proxy dialer: %w", err)
	}

	return rv.(Dialer), nil
}
