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

func getIP(ntw mtglib.Network, protocol string) net.IP {
	dialer := ntw.NativeDialer()
	client := ntw.MakeHTTPClient(func(ctx context.Context, network, address string) (essentials.Conn, error) {
		conn, err := dialer.DialContext(ctx, protocol, address)
		if err != nil {
			return nil, err
		}
		return essentials.WrapNetConn(conn), err
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
		resp.Body.Close()              //nolint: errcheck
	}()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	return net.ParseIP(strings.TrimSpace(string(data)))
}
