package network

import (
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	shadowsocks "github.com/shadowsocks/go-shadowsocks2/core"
	"golang.org/x/net/proxy"
)

type shadowsocksDialer struct {
	Dialer

	cipher shadowsocks.StreamConnCipher
}

func (s *shadowsocksDialer) Dial(network, address string) (net.Conn, error) {
	return s.DialContext(context.Background(), network, address)
}

func (s *shadowsocksDialer) DialContext(ctx context.Context,
	network, address string) (net.Conn, error) {
	conn, err := s.Dialer.DialContext(ctx, network, address)
	if err != nil {
		return nil, err // nolint: wrapcheck
	}

	return s.cipher.StreamConn(conn), nil
}

func NewShadowsocksDialer(proxyURL *url.URL,
	timeout time.Duration, bufferSize int) (Dialer, error) {
	username := proxyURL.User.Username()

	decoded, err := base64.RawURLEncoding.DecodeString(username)
	if err != nil {
		return nil, fmt.Errorf("cannot decode payload: %w", err)
	}

	chunks := strings.SplitN(string(decoded), ":", 2)
	if len(chunks) != 2 {
		return nil, fmt.Errorf("incorrect payload %s", username)
	}

	cipher, err := shadowsocks.PickCipher(chunks[0], nil, chunks[1])
	if err != nil {
		return nil, fmt.Errorf("cannot initialize shadowsocks cipher: %w", err)
	}

	socks5URL := *proxyURL
	socks5URL.Scheme = "socks5"
	socks5URL.User = nil

	dialer, err := NewDefaultDialer(timeout, bufferSize)
	if err != nil {
		return nil, fmt.Errorf("cannot initialize a base dialer: %w", err)
	}

	ssDialer := &shadowsocksDialer{
		Dialer: dialer,
		cipher: cipher,
	}

	rv, err := proxy.FromURL(&socks5URL, ssDialer)
	if err != nil {
		return nil, fmt.Errorf("cannot initialize ss proxy dialer: %w", err)
	}

	return rv.(Dialer), nil
}
