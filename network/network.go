package network

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/9seconds/mtg/v2/essentials"
	"github.com/9seconds/mtg/v2/mtglib"
)

type networkHTTPTransport struct {
	userAgent string
	next      http.RoundTripper
}

func (n networkHTTPTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", n.userAgent)

	return n.next.RoundTrip(req) //nolint: wrapcheck
}

type network struct {
	dialer      Dialer
	httpTimeout time.Duration
	userAgent   string
	dns         *dnsResolver
}

func (n *network) Dial(protocol, address string) (essentials.Conn, error) {
	return n.DialContext(context.Background(), protocol, address)
}

func (n *network) DialContext(ctx context.Context, protocol, address string) (essentials.Conn, error) {
	host, port, _ := net.SplitHostPort(address)

	ips, err := n.dnsResolve(protocol, host)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve dns names: %w", err)
	}

	rand.Shuffle(len(ips), func(i, j int) {
		ips[i], ips[j] = ips[j], ips[i]
	})

	var conn essentials.Conn

	for _, v := range ips {
		conn, err = n.dialer.DialContext(ctx, protocol, net.JoinHostPort(v, port))

		if err == nil {
			return conn, nil
		}
	}

	return nil, fmt.Errorf("cannot dial to %s:%s: %w", protocol, address, err)
}

func (n *network) MakeHTTPClient(dialFunc func(ctx context.Context,
	network, address string) (essentials.Conn, error),
) *http.Client {
	if dialFunc == nil {
		dialFunc = n.DialContext
	}

	return makeHTTPClient(n.userAgent, n.httpTimeout, dialFunc)
}

func (n *network) dnsResolve(protocol, address string) ([]string, error) {
	if net.ParseIP(address) != nil {
		return []string{address}, nil
	}

	ips := []string{}
	wg := &sync.WaitGroup{}
	mutex := &sync.Mutex{}

	switch protocol {
	case "tcp", "tcp4":
		wg.Add(1)

		go func() {
			defer wg.Done()

			resolved := n.dns.LookupA(address)

			mutex.Lock()
			ips = append(ips, resolved...)
			mutex.Unlock()
		}()
	}

	switch protocol {
	case "tcp", "tcp6":
		wg.Add(1)

		go func() {
			defer wg.Done()

			resolved := n.dns.LookupAAAA(address)

			mutex.Lock()
			ips = append(ips, resolved...)
			mutex.Unlock()
		}()
	}

	wg.Wait()

	if len(ips) == 0 {
		return nil, fmt.Errorf("cannot find any ips for %s:%s", protocol, address)
	}

	return ips, nil
}

// NewNetwork assembles an mtglib.Network compatible structure based on a
// dialer and given params.
//
// It brings simple DNS cache and DNS-Over-HTTPS when necessary.
func NewNetwork(dialer Dialer,
	userAgent, dohHostname string,
	httpTimeout time.Duration,
) (mtglib.Network, error) {
	switch {
	case httpTimeout < 0:
		return nil, fmt.Errorf("timeout should be positive number %s", httpTimeout)
	case httpTimeout == 0:
		httpTimeout = DefaultHTTPTimeout
	}

	if net.ParseIP(dohHostname) == nil {
		return nil, fmt.Errorf("hostname %s should be IP address", dohHostname)
	}

	return &network{
		dialer:      dialer,
		httpTimeout: httpTimeout,
		userAgent:   userAgent,
		dns: newDNSResolver(dohHostname,
			makeHTTPClient(userAgent, DNSTimeout, dialer.DialContext)),
	}, nil
}

func makeHTTPClient(userAgent string,
	timeout time.Duration,
	dialFunc func(ctx context.Context, network, address string) (essentials.Conn, error),
) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: networkHTTPTransport{
			userAgent: userAgent,
			next: &http.Transport{
				DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
					return dialFunc(ctx, network, address)
				},
			},
		},
	}
}
