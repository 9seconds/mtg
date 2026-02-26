package network

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/9seconds/mtg/v2/essentials"
)

type network struct {
	net.Dialer

	httpTimeout time.Duration
	idleTimeout time.Duration
	userAgent   string
}

func (n *network) Dial(network, address string) (essentials.Conn, error) {
	return n.DialContext(context.Background(), network, address)
}

func (n *network) DialContext(ctx context.Context, network, address string) (essentials.Conn, error) {
	switch network {
	case "tcp", "tcp4", "tcp6":
	default:
		return nil, fmt.Errorf("unsupported network %s", network)
	}

	conn, err := n.Dialer.DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}

	tcpConn := conn.(*net.TCPConn)

	return tcpConn, setCommonSocketOptions(tcpConn)
}

func (n *network) MakeHTTPClient(
	dialFunc func(context.Context, string, string) (essentials.Conn, error),
) *http.Client {
	if dialFunc == nil {
		dialFunc = n.DialContext
	}

	return &http.Client{
		Timeout: n.httpTimeout,
		Transport: networkHTTPTransport{
			userAgent: n.userAgent,
			next: &http.Transport{
				IdleConnTimeout: n.idleTimeout,
				DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
					return dialFunc(ctx, network, address)
				},
			},
		},
	}
}

func (n *network) NativeDialer() *net.Dialer {
	return &n.Dialer
}

func New(
	dnsResolver *net.Resolver,
	userAgent string,
	tcpTimeout,
	httpTimeout,
	idleTimeout time.Duration,
) Network {
	if dnsResolver == nil {
		dnsResolver = net.DefaultResolver
	}

	return &network{
		Dialer: net.Dialer{
			Timeout:       tcpTimeout,
			Resolver:      dnsResolver,
			FallbackDelay: -1,
		},
		userAgent:   userAgent,
		idleTimeout: idleTimeout,
		httpTimeout: httpTimeout,
	}
}
