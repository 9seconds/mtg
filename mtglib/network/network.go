package network

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"time"

	doh "github.com/babolivier/go-doh-client"
)

type Network struct {
	HTTP http.Client
	DNS  doh.Resolver

	dialer Dialer
}

func (d *Network) Dial(network, address string) (net.Conn, error) {
	return d.DialContext(context.Background(), network, address)
}

func (d *Network) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, _ := net.SplitHostPort(address)

	ips, err := d.resolveIPs(network, host)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve dns names: %w", err)
	}

	if len(ips) > 1 {
		rand.Shuffle(len(ips), func(i, j int) {
			ips[i], ips[j] = ips[j], ips[i]
		})
	}

	for _, v := range ips {
		if conn, err := d.dialer.DialContext(ctx, network, net.JoinHostPort(v, port)); err == nil {
			return conn, nil
		}
	}

	return nil, fmt.Errorf("cannot dial to %s:%s", network, address)
}

func (d *Network) resolveIPs(network, address string) ([]string, error) {
	if net.ParseIP(address) != nil {
		return []string{address}, nil
	}

	var ips []string

	switch network {
	case "tcp", "tcp4":
		if recs, _, err := d.DNS.LookupA(address); err == nil {
			for _, v := range recs {
				ips = append(ips, v.IP4)
			}
		}
	}

	switch network {
	case "tcp", "tcp6":
		if recs, _, err := d.DNS.LookupAAAA(address); err == nil {
			for _, v := range recs {
				ips = append(ips, v.IP6)
			}
		}
	}

	if len(ips) == 0 {
		return nil, fmt.Errorf("cannot find any ips for %s:%s", network, address)
	}

	return ips, nil
}

func NewNetwork(dialer Dialer, dohHostname string, httpTimeout time.Duration) (*Network, error) {
	switch {
	case httpTimeout < 0:
		return nil, fmt.Errorf("timeout should be positive number %v", httpTimeout)
	case httpTimeout == 0:
		httpTimeout = DefaultHTTPTimeout
	}

	if net.ParseIP(dohHostname) == nil {
		return nil, fmt.Errorf("hostname %s should be IP address", dohHostname)
	}

	dohHTTPClient := &http.Client{
		Timeout: httpTimeout,
		Transport: &http.Transport{
			DialContext: dialer.DialContext,
		},
	}
	network := &Network{
		dialer: dialer,
		DNS: doh.Resolver{
			Host:       dohHostname,
			Class:      doh.IN,
			HTTPClient: dohHTTPClient,
		},
	}
	network.HTTP = http.Client{
		Timeout: httpTimeout,
		Transport: &http.Transport{
			DialContext: network.DialContext,
		},
	}

	return network, nil
}
