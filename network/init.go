package network

import (
	"context"
	"errors"
	"net"
	"time"
)

const (
	DefaultTimeout     = 10 * time.Second
	DefaultIdleTimeout = time.Minute
	DefaultHTTPTimeout = 10 * time.Second
	DefaultBufferSize  = 4096

	ProxyDialerOpenThreshold        = 5
	ProxyDialerHalfOpenTimeout      = time.Minute
	ProxyDialerResetFailuresTimeout = 10 * time.Second

	DefaultDOHHostname = "9.9.9.9"
	DNSTimeout         = 5 * time.Second
)

var (
	ErrCircuitBreakerOpened     = errors.New("circuit breaker is opened")
	ErrCannotDialWithAllProxies = errors.New("cannot dial with all proxies")
)

type Dialer interface {
	Dial(network, address string) (net.Conn, error)
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}
