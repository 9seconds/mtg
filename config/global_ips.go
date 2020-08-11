package config

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"
)

const (
	ifconfigAddress = "https://ifconfig.co/ip"
	ifconfigTimeout = 10 * time.Second
)

func getGlobalIPv4(ctx context.Context) (net.IP, error) {
	ip, err := fetchIP(ctx, "tcp4")
	if err != nil || ip.To4() == nil {
		return nil, fmt.Errorf("cannot find public ipv4 address: %w", err)
	}

	return ip, nil
}

func getGlobalIPv6(ctx context.Context) (net.IP, error) {
	ip, err := fetchIP(ctx, "tcp6")
	if err != nil || ip.To4() != nil {
		return nil, fmt.Errorf("cannot find public ipv6 address: %w", err)
	}

	return ip, nil
}

func fetchIP(ctx context.Context, network string) (net.IP, error) {
	dialer := &net.Dialer{FallbackDelay: -1}
	client := &http.Client{
		Jar:     nil,
		Timeout: ifconfigTimeout,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, addr string) (net.Conn, error) {
				return dialer.DialContext(ctx, network, addr)
			},
		},
	}

	req, err := http.NewRequest("GET", ifconfigAddress, nil)
	if err != nil {
		return nil, fmt.Errorf("cannot create a request: %w", err)
	}

	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		if resp != nil {
			io.Copy(ioutil.Discard, resp.Body) // nolint: errcheck
		}

		return nil, fmt.Errorf("cannot perform a request: %w", err)
	}

	defer resp.Body.Close()

	respDataBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("cannot read response body: %w", err)
	}

	respData := strings.TrimSpace(string(respDataBytes))

	ip := net.ParseIP(respData)
	if ip == nil {
		return nil, fmt.Errorf("ifconfig.co returns incorrect IP %s", respData)
	}

	return ip, nil
}
