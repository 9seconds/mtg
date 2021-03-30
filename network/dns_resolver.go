package network

import (
	"net/http"
	"time"

	doh "github.com/babolivier/go-doh-client"
	"github.com/dgraph-io/ristretto"
)

const (
	dnsResolverSize     = 1024 * 1024 // 1mb
	dnsResolverKeepTime = 10 * time.Minute
)

type dnsResolver struct {
	resolver doh.Resolver
	cache    *ristretto.Cache
}

func (d dnsResolver) LookupA(hostname string) []string {
	key := "\x00." + hostname

	if value, ok := d.cache.Get(key); ok {
		return value.([]string)
	}

	var ips []string

	if recs, _, err := d.resolver.LookupA(hostname); err == nil {
		for _, v := range recs {
			ips = append(ips, v.IP4)
		}

		d.cache.SetWithTTL(key, ips, 0, dnsResolverKeepTime)
	}

	return ips
}

func (d dnsResolver) LookupAAAA(hostname string) []string {
	key := "\x01." + hostname

	if value, ok := d.cache.Get(key); ok {
		return value.([]string)
	}

	var ips []string

	if recs, _, err := d.resolver.LookupAAAA(hostname); err == nil {
		for _, v := range recs {
			ips = append(ips, v.IP6)
		}

		d.cache.SetWithTTL(key, ips, 0, dnsResolverKeepTime)
	}

	return ips
}

func newDNSResolver(hostname string, httpClient *http.Client) dnsResolver {
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 10 * dnsResolverSize, // nolint: gomnd // taken from official doc as a best practice value
		MaxCost:     dnsResolverSize,
		BufferItems: 64, // nolint: gomnd // taken from official doc as a best practice value
		Cost: func(value interface{}) int64 {
			var cost int64

			for _, v := range value.([]string) {
				cost += int64(len([]byte(v)))
			}

			return cost
		},
	})
	if err != nil {
		panic(err)
	}

	return dnsResolver{
		resolver: doh.Resolver{
			Host:       hostname,
			Class:      doh.IN,
			HTTPClient: httpClient,
		},
		cache: cache,
	}
}
