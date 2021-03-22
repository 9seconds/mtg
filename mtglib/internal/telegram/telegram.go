package telegram

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"strings"
)

type Telegram struct {
	dialer   Dialer
	preferIP preferIP
}

func (t Telegram) Dial(ctx context.Context, dc int) (net.Conn, error) {
	if dc < 0 || dc > 4 {
		return nil, fmt.Errorf("do not know how to dial to %d", dc)
	}

	var addresses []tgAddr

	if t.preferIP == preferIPOnlyIPv6 {
		addresses = []tgAddr{v6Addresses[dc]}
	} else {
		addresses = append(addresses, v4Addresses[dc]...)
		rand.Shuffle(len(addresses), func(i, j int) {
			addresses[i], addresses[j] = addresses[j], addresses[i]
		})
	}

	switch t.preferIP {
	case preferIPPreferIPv4:
		addresses = append(addresses, v6Addresses[dc])
	case preferIPPreferIPv6:
		addresses = append([]tgAddr{v6Addresses[dc]}, addresses...)
	case preferIPOnlyIPv4, preferIPOnlyIPv6:
	}

	var (
		conn net.Conn
		err  error
	)

	for _, v := range addresses {
		conn, err = t.dialer.DialContext(ctx, v.network, v.address)
		if err == nil {
			return conn, nil
		}
	}

	return nil, fmt.Errorf("cannot dial to %d dc: %w", dc, err)
}

func New(dialer Dialer, ipPreference string) (*Telegram, error) {
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

	return &Telegram{
		dialer:   dialer,
		preferIP: pref,
	}, nil
}
