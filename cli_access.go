package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/9seconds/mtg/v2/mtglib/network"
)

type runAccessResponse struct {
	IPv4   *runAccessResponseURLs `json:"ipv4,omitempty"`
	IPv6   *runAccessResponseURLs `json:"ipv6,omitempty"`
	Secret struct {
		Hex    string `json:"hex"`
		Base64 string `json:"base64"`
	} `json:"secret"`
}

type runAccessResponseURLs struct {
	IP        net.IP `json:"ip"`
	TgURL     string `json:"tg_url"`
	TgQrCode  string `json:"tg_qrcode"`
	TmeURL    string `json:"tme_url"`
	TmeQrCode string `json:"tme_qrcode"`
}

func runAccess(cli *CLI) {
	filefp, err := os.Open(cli.Access.ConfigPath)
	if err != nil {
		exit(err)
	}

	defer filefp.Close()

	conf, err := parseConfig(filefp)
	if err != nil {
		exit(err)
	}

	ntw, err := makeNetwork(conf)
	if err != nil {
		exit(err)
	}

	ipv4 := conf.Network.PublicIP.IPv4.Value(nil)
	ipv6 := conf.Network.PublicIP.IPv6.Value(nil)

	if ipv4 == nil {
		ipv4 = runAccessGetIP(ntw, "tcp4")
	}

	if ipv6 == nil {
		ipv6 = runAccessGetIP(ntw, "tcp6")
	}

	resp := runAccessResponse{
		IPv4: runMakeAccessResponseURLs(ipv4, conf, cli),
		IPv6: runMakeAccessResponseURLs(ipv6, conf, cli),
	}
	resp.Secret.Base64 = conf.Secret.Base64()
	resp.Secret.Hex = conf.Secret.Hex()

	encoder := json.NewEncoder(os.Stdout)

	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(resp); err != nil {
		exit(err)
	}
}

func runAccessGetIP(ntw *network.Network, protocol string) net.IP {
	client := &http.Client{
		Timeout: ntw.HTTP.Timeout,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
				return ntw.DialContext(ctx, protocol, address)
			},
		},
	}

	resp, err := client.Get("https://ifconfig.co") // nolint: bodyclose, noctx
	if err != nil {
		return nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	defer exhaustResponse(resp)

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	return net.ParseIP(strings.TrimSpace(string(data)))
}

func runMakeAccessResponseURLs(ip net.IP, conf *config, cli *CLI) *runAccessResponseURLs {
	if ip == nil {
		return nil
	}

	values := url.Values{}

	values.Set("server", ip.String())
	values.Set("port", strconv.Itoa(int(conf.BindTo.port.Value(0))))

	if cli.Access.Hex {
		values.Set("secret", conf.Secret.Hex())
	} else {
		values.Set("secret", conf.Secret.Base64())
	}

	urlQuery := values.Encode()

	rv := &runAccessResponseURLs{
		IP: ip,
		TgURL: (&url.URL{
			Scheme:   "tg",
			Host:     "proxy",
			RawQuery: urlQuery,
		}).String(),
		TmeURL: (&url.URL{
			Scheme:   "https",
			Host:     "t.me",
			Path:     "proxy",
			RawQuery: urlQuery,
		}).String(),
	}

	rv.TgQrCode = runMakeAccessResponseURLsQRCode(rv.TgURL)
	rv.TmeQrCode = runMakeAccessResponseURLsQRCode(rv.TmeURL)

	return rv
}

func runMakeAccessResponseURLsQRCode(data string) string {
	values := url.Values{}

	values.Set("qzone", "4")
	values.Set("format", "svg")
	values.Set("data", data)

	return (&url.URL{
		Scheme:   "https",
		Host:     "api.qrserver.com",
		Path:     "v1/create-qr-code",
		RawQuery: values.Encode(),
	}).String()
}
