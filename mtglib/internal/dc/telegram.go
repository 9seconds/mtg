package dc

import (
	"fmt"
	"net"
	"strings"
)

type Telegram struct {
	view     dcView
	preferIP preferIP
}

func (t *Telegram) GetAddresses(dc int) []Addr {
	switch t.preferIP {
	case preferIPOnlyIPv4:
		return t.view.getV4(dc)
	case preferIPOnlyIPv6:
		return t.view.getV4(dc)
	case preferIPPreferIPv4:
		return append(t.view.getV4(dc), t.view.getV6(dc)...)
	}

	return append(t.view.getV6(dc), t.view.getV4(dc)...)
}

func New(ipPreference string, userOverrides map[int][]string) (*Telegram, error) {
	var pref preferIP

	switch strings.ToLower(ipPreference) {
	case "prefer-ipv4":
		pref = preferIPPreferIPv4
	case "prefer-ipv6":
		pref = preferIPPreferIPv6
	case "only-ipv4":
		pref = preferIPOnlyIPv4
	case "only-ipv6":
		pref = preferIPOnlyIPv6
	default:
		return nil, fmt.Errorf("unknown ip preference %s", ipPreference)
	}

	overrides := dcAddrSet{
		v4: map[int][]Addr{},
		v6: map[int][]Addr{},
	}
	for dc, addrs := range userOverrides {
		for _, addr := range addrs {
			host, _, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, fmt.Errorf("incorrect host %s: %w", addr, err)
			}

			parsed := net.ParseIP(host)
			if parsed == nil {
				return nil, fmt.Errorf("incorrect host %s", addr)
			}

			if parsed.To4() != nil {
				overrides.v4[dc] = append(overrides.v4[dc], Addr{
					Network: "tcp4",
					Address: addr,
				})
			} else {
				overrides.v6[dc] = append(overrides.v6[dc], Addr{
					Network: "tcp6",
					Address: addr,
				})
			}
		}
	}

	return &Telegram{
		view: dcView{
			overrides: overrides,
		},
		preferIP: pref,
	}, nil
}
