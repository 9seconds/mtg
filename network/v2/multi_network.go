package network

import (
	"context"
	"errors"
	"math/rand"
	"net"
	"net/http"

	"github.com/9seconds/mtg/v2/essentials"
)

type multiNetwork struct {
	networks []Network
}

func (m multiNetwork) Dial(network, address string) (essentials.Conn, error) {
	return m.DialContext(context.Background(), network, address)
}

func (m multiNetwork) DialContext(ctx context.Context, network, address string) (essentials.Conn, error) {
	networks := m.networks

	if len(networks) > 1 {
		networks = make([]Network, len(m.networks))
		copy(networks, m.networks)

		rand.Shuffle(len(m.networks), func(i, j int) {
			networks[i], networks[j] = networks[j], networks[i]
		})
	}

	errs := make([]error, 1, len(networks)+1)
	errs[0] = ErrCannotDial

	for _, ntw := range networks {
		conn, err := ntw.DialContext(ctx, network, address)
		if err == nil {
			return conn, nil
		}

		errs = append(errs, err)
	}

	return nil, errors.Join(errs...)
}

func (m multiNetwork) NativeDialer() *net.Dialer {
	return m.networks[0].NativeDialer()
}

func (m multiNetwork) MakeHTTPClient(
	dialFunc func(context.Context, string, string) (essentials.Conn, error),
) *http.Client {
	if dialFunc == nil {
		dialFunc = m.DialContext
	}

	return m.networks[0].MakeHTTPClient(dialFunc)
}

func Join(networks ...Network) (Network, error) {
	if len(networks) == 0 {
		return nil, errors.New("cannot join no networks")
	}

	return multiNetwork{
		networks: networks,
	}, nil
}
