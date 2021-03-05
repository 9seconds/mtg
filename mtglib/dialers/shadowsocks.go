package dialers

import (
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	shadowsocks "github.com/shadowsocks/go-shadowsocks2/core"
)

type shadowsocksBaseDialer struct {
	base   BaseDialer
	cipher shadowsocks.StreamConnCipher
}

func (s *shadowsocksBaseDialer) Dial(network, address string) (net.Conn, error) {
	conn, err := s.base.Dial(network, address)
	if err != nil {
		return nil, err
	}

	return s.cipher.StreamConn(conn), nil
}

func (s *shadowsocksBaseDialer) DialContext(ctx context.Context,
	network, address string) (net.Conn, error) {
	conn, err := s.base.DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}

	return s.cipher.StreamConn(conn), nil
}

func NewShadowsocksBaseDialer(proxyUrl *url.URL,
	timeout time.Duration, bufferSize int) (BaseDialer, error) {
	username := proxyUrl.User.Username()

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

	baseDialer, err := NewDefaultBaseDialer(timeout, bufferSize)
	if err != nil {
		return nil, fmt.Errorf("cannot initialize a base dialer: %w", err)

	}

	return &shadowsocksBaseDialer{
		base:   baseDialer,
		cipher: cipher,
	}, nil
}
