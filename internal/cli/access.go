package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/9seconds/mtg/v2/essentials"
	"github.com/9seconds/mtg/v2/internal/config"
	"github.com/9seconds/mtg/v2/internal/utils"
	"github.com/9seconds/mtg/v2/mtglib"
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
	TgURL     string `json:"tg_url"`     //nolint: tagliatelle
	TgQrCode  string `json:"tg_qrcode"`  //nolint: tagliatelle
	TmeURL    string `json:"tme_url"`    //nolint: tagliatelle
	TmeQrCode string `json:"tme_qrcode"` //nolint: tagliatelle
}

type Access struct {
	ConfigPath string `kong:"arg,required,type='existingfile',help='Path to the configuration file.',name='config-path'"`                 //nolint: lll
	PublicIPv4 net.IP `kong:"help='Public IPv4 address for proxy. By default it is resolved via remote website',name='ipv4',short='i'"`   //nolint: lll
	PublicIPv6 net.IP `kong:"help='Public IPv6 address for proxy. By default it is resolved via remote website',name='ipv6',short='I'"`   //nolint: lll
	Port       uint   `kong:"help='Port number. Default port is taken from configuration file, bind-to parameter',type:'uint',short='p'"` //nolint: lll
	Hex        bool   `kong:"help='Print secret in hex encoding.',short='x'"`
}

func (a *Access) Run(cli *CLI, version string) error {
	conf, err := utils.ReadConfig(a.ConfigPath)
	if err != nil {
		return fmt.Errorf("cannot init config: %w", err)
	}

	resp := &accessResponse{}
	resp.Secret.Base64 = conf.Secret.Base64()
	resp.Secret.Hex = conf.Secret.Hex()

	ntw, err := makeNetwork(conf, version)
	if err != nil {
		return fmt.Errorf("cannot init network: %w", err)
	}

	wg := &sync.WaitGroup{}
	wg.Add(2) //nolint: gomnd

	go func() {
		defer wg.Done()

		ip := a.PublicIPv4
		if ip == nil {
			ip = a.getIP(ntw, "tcp4")
		}

		if ip != nil {
			ip = ip.To4()
		}

		resp.IPv4 = a.makeURLs(conf, ip)
	}()

	go func() {
		defer wg.Done()

		ip := a.PublicIPv6
		if ip == nil {
			ip = a.getIP(ntw, "tcp6")
		}

		if ip != nil {
			ip = ip.To16()
		}

		resp.IPv6 = a.makeURLs(conf, ip)
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

func (a *Access) getIP(ntw mtglib.Network, protocol string) net.IP {
	client := ntw.MakeHTTPClient(func(ctx context.Context, network, address string) (essentials.Conn, error) {
		return ntw.DialContext(ctx, protocol, address) //nolint: wrapcheck
	})

	req, err := http.NewRequest(http.MethodGet, "https://ifconfig.co", nil) //nolint: noctx
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
		io.Copy(io.Discard, resp.Body) //nolint: errcheck
		resp.Body.Close()
	}()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	return net.ParseIP(strings.TrimSpace(string(data)))
}

func (a *Access) makeURLs(conf *config.Config, ip net.IP) *accessResponseURLs {
	if ip == nil {
		return nil
	}

	portNo := a.Port
	if portNo == 0 {
		portNo = conf.BindTo.Port
	}

	values := url.Values{}
	values.Set("server", ip.String())
	values.Set("port", strconv.Itoa(int(portNo)))

	if a.Hex {
		values.Set("secret", conf.Secret.Hex())
	} else {
		values.Set("secret", conf.Secret.Base64())
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
	rv.TgQrCode = utils.MakeQRCodeURL(rv.TgURL)
	rv.TmeQrCode = utils.MakeQRCodeURL(rv.TmeURL)

	return rv
}
