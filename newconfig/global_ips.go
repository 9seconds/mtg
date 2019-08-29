package newconfig

import (
	"context"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/juju/errors"
)

const (
	ifconfigAddress = "https://ifconfig.co/ip"
	ifconfigTimeout = 10 * time.Second
)

func getGlobalIPv4() (net.IP, error) {
	ip, err := fetchIP("tcp4")
	if err != nil || ip.To4() == nil {
		return nil, errors.Annotate(err, "Cannot find public ipv4 address")
	}
	return ip, nil
}

func getGlobalIPv6() (net.IP, error) {
	ip, err := fetchIP("tcp6")
	if err != nil || ip.To4() != nil {
		return nil, errors.Annotate(err, "Cannot find public ipv6 address")
	}
	return ip, nil
}

func fetchIP(network string) (net.IP, error) {
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

	resp, err := client.Get(ifconfigAddress)
	if err != nil {
		if resp != nil {
			io.Copy(ioutil.Discard, resp.Body) // nolint: errcheck
		}
		return nil, errors.Annotate(err, "Cannot perform a request")
	}
	defer resp.Body.Close() // nolint: errcheck

	respDataBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Annotate(err, "Cannot read response body")
	}
	respData := strings.TrimSpace(string(respDataBytes))

	ip := net.ParseIP(respData)
	if ip == nil {
		return nil, errors.Errorf("ifconfig.co returns incorrect IP %s", respData)
	}

	return ip, nil
}
