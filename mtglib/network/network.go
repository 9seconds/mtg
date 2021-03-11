package network

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"sync"
	"time"

	doh "github.com/babolivier/go-doh-client"
)

type network struct {
	idleTimeout time.Duration
	dialer      Dialer
	dns         doh.Resolver
}

func (n *network) Dial(protocol, address string) (net.Conn, error) {
	return n.DialContext(context.Background(), protocol, address)
}

func (n *network) DialContext(ctx context.Context, protocol, address string) (net.Conn, error) {
	host, port, _ := net.SplitHostPort(address)

	ips, err := n.DNSResolve(protocol, host)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve dns names: %w", err)
	}

	if len(ips) > 1 {
		rand.Shuffle(len(ips), func(i, j int) {
			ips[i], ips[j] = ips[j], ips[i]
		})
	}

	var conn net.Conn

	for _, v := range ips {
		conn, err = n.dialer.DialContext(ctx, protocol, net.JoinHostPort(v, port))

		if err == nil {
			return conn, nil
		}
	}

	return nil, fmt.Errorf("cannot dial to %s:%s: %w", protocol, address, err)
}

func (n *network) DNSResolve(protocol, address string) ([]string, error) {
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

			if recs, _, err := n.dns.LookupA(address); err == nil {
				mutex.Lock()
				defer mutex.Unlock()

				for _, v := range recs {
					ips = append(ips, v.IP4)
				}
			}
		}()
	}

	switch protocol {
	case "tcp", "tcp6":
		wg.Add(1)

		go func() {
			defer wg.Done()

			if recs, _, err := n.dns.LookupAAAA(address); err == nil {
				mutex.Lock()
				defer mutex.Unlock()

				for _, v := range recs {
					ips = append(ips, v.IP6)
				}
			}
		}()
	}

	wg.Wait()

	if len(ips) == 0 {
		return nil, fmt.Errorf("cannot find any ips for %s:%s", protocol, address)
	}

	return ips, nil
}

func (n *network) IdleTimeout() time.Duration {
	return n.idleTimeout
}

func (n *network) MakeHTTPClient(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = DefaultHTTPTimeout
	}

	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			DialContext: n.DialContext,
		},
	}
}

func NewNetwork(dialer Dialer, dohHostname string, idleTimeout time.Duration) (Network, error) {
	switch {
	case idleTimeout < 0:
		return nil, fmt.Errorf("timeout should be positive number %s", idleTimeout)
	case idleTimeout == 0:
		idleTimeout = DefaultIdleTimeout
	}

	if net.ParseIP(dohHostname) == nil {
		return nil, fmt.Errorf("hostname %s should be IP address", dohHostname)
	}

	return &network{
		dialer:      dialer,
		idleTimeout: idleTimeout,
		dns: doh.Resolver{
			Host:  dohHostname,
			Class: doh.IN,
			HTTPClient: &http.Client{
				Timeout: DNSTimeout,
				Transport: &http.Transport{
					DialContext: dialer.DialContext,
				},
			},
		},
	}, nil
}
