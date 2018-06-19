package config

import (
	"io/ioutil"
	"net"
	"net/http"
	"strings"

	"github.com/juju/errors"
)

func getGlobalIPv4() (net.IP, error) {
	return fetchIP("https://v4.ifconfig.co/ip")
}

func getGlobalIPv6() (net.IP, error) {
	return fetchIP("https://v6.ifconfig.co/ip")
}

func fetchIP(url string) (net.IP, error) {
	resp, err := http.Get(url)
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
