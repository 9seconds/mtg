package cli

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/9seconds/mtg/v2/essentials"
	"github.com/9seconds/mtg/v2/mtglib"
)

// publicIPEndpoints are tried in order. Each must return the client's public
// IP as a single address in the plain-text response body.
var publicIPEndpoints = []string{
	"https://ifconfig.co",
	"https://icanhazip.com",
	"https://ifconfig.me",
}

func getIP(ntw mtglib.Network, protocol string) net.IP {
	dialer := ntw.NativeDialer()
	client := ntw.MakeHTTPClient(func(ctx context.Context, network, address string) (essentials.Conn, error) {
		conn, err := dialer.DialContext(ctx, protocol, address)
		if err != nil {
			return nil, err
		}
		return essentials.WrapNetConn(conn), err
	})

	for _, endpoint := range publicIPEndpoints {
		if ip := fetchPublicIP(client, endpoint); ip != nil {
			return ip
		}
	}

	return nil
}

func fetchPublicIP(client *http.Client, endpoint string) net.IP {
	req, err := http.NewRequest(http.MethodGet, endpoint, nil) //nolint: noctx
	if err != nil {
		return nil
	}

	req.Header.Set("Accept", "text/plain")
	req.Header.Set("User-Agent", "curl/8")

	resp, err := client.Do(req)
	if err != nil {
		return nil
	}

	defer func() {
		io.Copy(io.Discard, resp.Body) //nolint: errcheck
		resp.Body.Close()              //nolint: errcheck
	}()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	return net.ParseIP(strings.TrimSpace(string(data)))
}
