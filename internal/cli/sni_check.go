package cli

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/9seconds/mtg/v2/internal/config"
	"github.com/9seconds/mtg/v2/mtglib"
)

type sniCheckResult struct {
	ResolvedIP4 []string
	ResolvedIP6 []string
	OurIP4      string
	OurIP6      string
}

func runSNICheck(
	ctx context.Context,
	conf *config.Config,
	resolver *net.Resolver,
	ntw mtglib.Network,
) (sniCheckResult, error) {
	res := sniCheckResult{}

	addrs, err := resolver.LookupIPAddr(ctx, conf.Secret.Host)
	if err != nil {
		return res, fmt.Errorf("cannot resolve addresses of %s: %w", conf.Secret.Host, err)
	}

	if len(addrs) == 0 {
		return res, fmt.Errorf("no known addresses for %s", conf.Secret.Host)
	}

	for _, addr := range addrs {
		if ip := addr.IP.To4(); ip == nil {
			res.ResolvedIP6 = append(res.ResolvedIP6, addr.IP.To16().String())
		} else {
			res.ResolvedIP4 = append(res.ResolvedIP4, ip.String())
		}
	}

	wg := &sync.WaitGroup{}

	if len(res.ResolvedIP4) > 0 {
		wg.Go(func() {
			ip := conf.PublicIPv4.Get(nil)
			if ip == nil {
				ip, _ = getIP(ctx, ntw, "tcp4")
			}

			if ip != nil {
				res.OurIP4 = ip.To4().String()
			}
		})
	}

	if len(res.ResolvedIP6) > 0 {
		wg.Go(func() {
			ip := conf.PublicIPv6.Get(nil)
			if ip == nil {
				ip, _ = getIP(ctx, ntw, "tcp6")
			}

			if ip != nil {
				res.OurIP6 = ip.To16().String()
			}
		})
	}

	wg.Wait()

	return res, nil
}
