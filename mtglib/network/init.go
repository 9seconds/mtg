package network

import (
	"context"
	"errors"
	"net"
	"net/http"
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

type DialFunc func(ctx context.Context, protocol, address string) (net.Conn, error)

type Dialer interface {
	Dial(network, address string) (net.Conn, error)
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

type Network interface {
	Dialer

	DNSResolve(network, hostname string) (ips []string, err error)
	MakeHTTPClient(DialFunc) *http.Client
	IdleTimeout() time.Duration
}
