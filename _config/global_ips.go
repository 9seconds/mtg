package config

import (
	"context"
	"io/ioutil"
	"net"
	"net/http"
	"strings"

	"github.com/juju/errors"
)

const ifconfigAddress = "https://ifconfig.co/ip"

func getGlobalIPv4() (net.IP, error) {
	return fetchIP("tcp4")
}

func getGlobalIPv6() (net.IP, error) {
	return fetchIP("tcp6")
}

func fetchIP(network string) (net.IP, error) {
	dialer := &net.Dialer{FallbackDelay: -1}
	client := &http.Client{
		Jar: nil,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, addr string) (net.Conn, error) {
				return dialer.DialContext(ctx, network, addr)
			},
		},
	}

	resp, err := client.Get(ifconfigAddress)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() // nolint: errcheck

	respDataBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	respData := strings.TrimSpace(string(respDataBytes))

	ip := net.ParseIP(respData)
	if ip == nil {
		return nil, errors.Errorf("ifconfig.co returns incorrect IP %s", respData)
	}

	return ip, nil
}
