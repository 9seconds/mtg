package network

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/ncruces/go-dns"
)

var dnsCacheOptions = []dns.CacheOption{
	dns.MaxCacheEntries(dns.DefaultMaxCacheEntries),
	dns.MaxCacheTTL(time.Hour),
	dns.NegativeCache(false),
}

func GetDNS(u *url.URL) (*net.Resolver, error) {
	if u == nil {
		return dns.NewCachingResolver(nil, dnsCacheOptions...), nil
	}

	if u.Scheme == "" {
		u.Scheme = "udp"
	}
	if u.Scheme == "udp" && u.Host == "" {
		u.Host = u.Path
		u.Path = ""
	}

	switch u.Scheme {
	case "tls":
		return dns.NewDoTResolver(u.Host, dns.DoTCache(dnsCacheOptions...))
	case "https":
		if u.Path == "" {
			u.Path = "/dns-query"
		}

		return dns.NewDoHResolver(u.String(), dns.DoHCache(dnsCacheOptions...))
	case "udp", "":
	default:
		return nil, fmt.Errorf("unsupported DNS %v", u)
	}

	port := u.Port()
	if port == "" {
		port = "53"
	}

	hostport := net.JoinHostPort(u.Hostname(), port)
	dialer := &net.Dialer{}
	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			return dialer.DialContext(ctx, "udp", hostport)
		},
	}

	return dns.NewCachingResolver(resolver, dnsCacheOptions...), nil
}
