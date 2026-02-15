package telegram

import (
	"context"
	"net"
	"strconv"

	"github.com/gotd/td/telegram"
)

type rpcClient struct {
	*telegram.Client
}

func (r rpcClient) getDCAddresses(logger loggerInterface, ctx context.Context) (dcAddresses, error) {
	addrs := dcAddresses{
		v4: map[int][]tgAddr{},
		v6: map[int][]tgAddr{},
	}

	err := r.Client.Run(ctx, func(_ context.Context) error {
		for _, opt := range r.Client.Config().DCOptions {
			addr := net.JoinHostPort(opt.IPAddress, strconv.Itoa(opt.Port))

			if opt.Ipv6 {
				addrs.v6[opt.ID] = append(addrs.v6[opt.ID], tgAddr{
					network: "tcp6",
					address: addr,
				})
			} else {
				addrs.v4[opt.ID] = append(addrs.v4[opt.ID], tgAddr{
					network: "tcp4",
					address: addr,
				})
			}
		}

		return nil
	})

	return addrs, err
}
