package cli

import (
	"context"
	"net"
	"sync"

	"github.com/9seconds/mtg/v2/internal/config"
	"github.com/9seconds/mtg/v2/mtglib"
)

// sniCheckResult captures the outcome of comparing the secret hostname's DNS
// records with this server's public IP addresses.
//
// IPv4Match/IPv6Match are true when either a matching record was found, or
// when the corresponding public IP could not be detected — in which case
// there is nothing to compare against.
type sniCheckResult struct {
	Host       string
	Resolved   []net.IP
	OurIPv4    net.IP
	OurIPv6    net.IP
	IPv4Match  bool
	IPv6Match  bool
	ResolveErr error
}

// Known reports whether at least one public IP family was detected.
func (r sniCheckResult) Known() bool {
	return r.OurIPv4 != nil || r.OurIPv6 != nil
}

// OK reports whether the check produced a clean result: the hostname was
// resolved, at least one public IP family is known, and every known family
// matches a resolved record.
func (r sniCheckResult) OK() bool {
	if r.Host == "" {
		return true
	}

	if r.ResolveErr != nil || !r.Known() {
		return false
	}

	return r.IPv4Match && r.IPv6Match
}

// runSNICheck resolves conf.Secret.Host and compares the result with the
// server's public IPv4 and IPv6. Public IPs come from config first and fall
// back to on-the-fly detection via ntw. IP detection for the two families
// runs concurrently.
func runSNICheck(ctx context.Context,
	resolver *net.Resolver,
	conf *config.Config,
	ntw mtglib.Network,
) sniCheckResult {
	res := sniCheckResult{Host: conf.Secret.Host}

	if res.Host == "" {
		res.IPv4Match = true
		res.IPv6Match = true

		return res
	}

	addrs, err := resolver.LookupIPAddr(ctx, res.Host)
	if err != nil {
		res.ResolveErr = err

		return res
	}

	res.Resolved = make([]net.IP, 0, len(addrs))
	for _, a := range addrs {
		res.Resolved = append(res.Resolved, a.IP)
	}

	wg := sync.WaitGroup{}

	wg.Go(func() {
		res.OurIPv4 = conf.PublicIPv4.Get(nil)
		if res.OurIPv4 == nil {
			res.OurIPv4 = getIP(ntw, "tcp4")
		}
	})

	wg.Go(func() {
		res.OurIPv6 = conf.PublicIPv6.Get(nil)
		if res.OurIPv6 == nil {
			res.OurIPv6 = getIP(ntw, "tcp6")
		}
	})

	wg.Wait()

	res.IPv4Match = res.OurIPv4 == nil
	res.IPv6Match = res.OurIPv6 == nil

	for _, ip := range res.Resolved {
		if res.OurIPv4 != nil && ip.String() == res.OurIPv4.String() {
			res.IPv4Match = true
		}

		if res.OurIPv6 != nil && ip.String() == res.OurIPv6.String() {
			res.IPv6Match = true
		}
	}

	return res
}
