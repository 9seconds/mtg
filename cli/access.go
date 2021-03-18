package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
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
	Port      uint   `json:"port"`
	TgURL     string `json:"tg_url"`
	TgQrCode  string `json:"tg_qrcode"`
	TmeURL    string `json:"tme_url"`
	TmeQrCode string `json:"tme_qrcode"`
}

type Access struct {
	base

	PublicIPv4 net.IP `kong:"help='Public IPv4 address for proxy. By default it is resolved via remote website',name='ipv4',short='i'"`   // nolint: lll
	PublicIPv6 net.IP `kong:"help='Public IPv6 address for proxy. By default it is resolved via remote website',name='ipv6',short='I'"`   // nolint: lll
	Port       uint   `kong:"help='Port number. Default port is taken from configuration file, bind-to parameter',type:'uint',short='p'"` // nolint: lll
	Hex        bool   `kong:"help='Print secret in hex encoding.',short='x'"`
}

func (c *Access) Run(cli *CLI, version string) error {
	if err := c.ReadConfig(version); err != nil {
		return fmt.Errorf("cannot init config: %w", err)
	}

	return c.Execute(cli)
}

func (c *Access) Execute(cli *CLI) error {
	resp := &accessResponse{}
	resp.Secret.Base64 = c.Config.Secret.Base64()
	resp.Secret.Hex = c.Config.Secret.Hex()

	wg := &sync.WaitGroup{}
	wg.Add(2) // nolint: gomnd

	go func() {
		defer wg.Done()

		ip := cli.Access.PublicIPv4
		if ip == nil {
			ip = c.getIP("tcp4")
		}

		if ip != nil {
			ip = ip.To4()
		}

		resp.IPv4 = c.makeURLs(ip, cli)
	}()

	go func() {
		defer wg.Done()

		ip := cli.Access.PublicIPv6
		if ip == nil {
			ip = c.getIP("tcp6")
		}

		if ip != nil {
			ip = ip.To16()
		}

		resp.IPv6 = c.makeURLs(ip, cli)
	}()

	wg.Wait()

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(resp); err != nil {
		return fmt.Errorf("cannot dump access json: %w", err)
	}

	return nil
}

func (c *Access) getIP(protocol string) net.IP {
	client := c.Network.MakeHTTPClient(func(ctx context.Context, network, address string) (net.Conn, error) {
		return c.Network.DialContext(ctx, protocol, address)
	})

	req, err := http.NewRequest(http.MethodGet, "https://ifconfig.co", nil) // nolint: noctx
	if err != nil {
		panic(err)
	}

	req.Header.Add("Accept", "text/plain")

	resp, err := client.Do(req)
	if err != nil {
		return nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	defer func() {
		io.Copy(ioutil.Discard, resp.Body) // nolint: errcheck
		resp.Body.Close()
	}()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	return net.ParseIP(strings.TrimSpace(string(data)))
}

func (c *Access) makeURLs(ip net.IP, cli *CLI) *accessResponseURLs {
	if ip == nil {
		return nil
	}

	portNo := cli.Access.Port
	if portNo == 0 {
		portNo = c.Config.BindTo.PortValue(0)
	}

	values := url.Values{}
	values.Set("server", ip.String())
	values.Set("port", strconv.Itoa(int(portNo)))

	if cli.Access.Hex {
		values.Set("secret", c.Config.Secret.Hex())
	} else {
		values.Set("secret", c.Config.Secret.Base64())
	}

	urlQuery := values.Encode()

	rv := &accessResponseURLs{
		IP:   ip,
		Port: portNo,
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

func (c *Access) makeQRCode(data string) string {
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
