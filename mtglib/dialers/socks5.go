package dialers

import (
	"fmt"
	"net/url"
	"time"

	"golang.org/x/net/proxy"
)

func NewSocks5BaseDialer(proxyUrl *url.URL, timeout time.Duration, bufferSize int) (BaseDialer, error) {
	baseDialer, err := NewDefaultBaseDialer(timeout, bufferSize)
	if err != nil {
		return nil, fmt.Errorf("cannot initialize base dialer: %w", err)
	}

	rv, err := proxy.FromURL(proxyUrl, baseDialer.(*defaultBaseDialer))
	if err != nil {
		return nil, fmt.Errorf("cannot initialize socks5 proxy dialer: %w", err)
	}

	return rv.(BaseDialer), nil
}
