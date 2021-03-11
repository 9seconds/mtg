package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type accessResponse struct {
	IPv4   *accessResponseURLs `json:"ipv4,omitempty"`
	IPv6   *accessResponseURLs `json:"ipv6,omitempty"`
	Secret struct {
		Hex    string `json:"hex"`
		Base64 string `json:"base64"`
	} `json:"secret"`
}

type accessResponseURLs struct {
	IP        net.IP `json:"ip"`
	TgURL     string `json:"tg_url"`
	TgQrCode  string `json:"tg_qrcode"`
	TmeURL    string `json:"tme_url"`
	TmeQrCode string `json:"tme_qrcode"`
}

type cliCommandAccess struct {
	cli

	ConfigPath string `arg required type:"existingfile" help:"Path to the configuration file." name:"config-path"` // nolint: lll, govet
	Hex        bool   `help:"Print secret in hex encoding."`
}

func (c *cliCommandAccess) Run(cli *CLI) error {
	if err := c.ReadConfig(cli.Access.ConfigPath); err != nil {
		return fmt.Errorf("cannot init config: %w", err)
	}

	ipv4 := c.conf.Network.PublicIP.IPv4.Value(nil)
	ipv6 := c.conf.Network.PublicIP.IPv6.Value(nil)

	if ipv4 == nil {
		ipv4 = c.getIP("tcp4")
	}

	if ipv6 == nil {
		ipv6 = c.getIP("tcp6")
	}

	resp := accessResponse{
		IPv4: c.makeURLs(ipv4, cli),
		IPv6: c.makeURLs(ipv6, cli),
	}
	resp.Secret.Base64 = c.conf.Secret.Base64()
	resp.Secret.Hex = c.conf.Secret.Hex()

	encoder := json.NewEncoder(os.Stdout)

	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(resp); err != nil {
		return fmt.Errorf("cannot dump access json: %w", err)
	}

	return nil
}

func (c *cliCommandAccess) getIP(protocol string) net.IP {
	client := c.network.MakeHTTPClient(0)
	client.Transport = &http.Transport{
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			return c.network.DialContext(ctx, protocol, address)
		},
	}

	c.network.PatchHTTPClient(client)

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

func (c *cliCommandAccess) makeURLs(ip net.IP, cli *CLI) *accessResponseURLs {
	if ip == nil {
		return nil
	}

	values := url.Values{}

	values.Set("server", ip.String())
	values.Set("port", strconv.Itoa(int(c.conf.BindTo.PortValue(0))))

	if cli.Access.Hex {
		values.Set("secret", c.conf.Secret.Hex())
	} else {
		values.Set("secret", c.conf.Secret.Base64())
	}

	urlQuery := values.Encode()

	rv := &accessResponseURLs{
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

	rv.TgQrCode = c.makeQRCode(rv.TgURL)
	rv.TmeQrCode = c.makeQRCode(rv.TmeURL)

	return rv
}

func (c *cliCommandAccess) makeQRCode(data string) string {
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
