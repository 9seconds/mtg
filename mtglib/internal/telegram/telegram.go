package telegram

import (
	"context"
	"fmt"
	"strings"

	"github.com/9seconds/mtg/v2/essentials"
)

type Telegram struct {
	dialer   Dialer
	preferIP preferIP
	pool     addressPool
}

func (t Telegram) Dial(ctx context.Context, dc int) (essentials.Conn, error) {
	var addresses []tgAddr

	switch t.preferIP {
	case preferIPOnlyIPv4:
		addresses = t.pool.getV4(dc)
	case preferIPOnlyIPv6:
		addresses = t.pool.getV6(dc)
	case preferIPPreferIPv4:
		addresses = append(t.pool.getV4(dc), t.pool.getV6(dc)...)
	case preferIPPreferIPv6:
		addresses = append(t.pool.getV6(dc), t.pool.getV4(dc)...)
	}

	var conn essentials.Conn

	err := errNoAddresses

	for _, v := range addresses {
		conn, err = t.dialer.DialContext(ctx, v.network, v.address)
		if err == nil {
			return conn, nil
		}
	}

	return nil, fmt.Errorf("cannot dial to %d dc: %w", dc, err)
}

func (t Telegram) IsKnownDC(dc int) bool {
	return t.pool.isValidDC(dc)
}

func (t Telegram) GetFallbackDC() int {
	return t.pool.getRandomDC()
}

func New(dialer Dialer, ipPreference string, useTestDCs bool) (*Telegram, error) {
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

	pool := addressPool{
		v4: productionV4Addresses,
		v6: productionV6Addresses,
	}
	if useTestDCs {
		pool.v4 = testV4Addresses
		pool.v6 = testV6Addresses
	}

	return &Telegram{
		dialer:   dialer,
		preferIP: pref,
		pool:     pool,
	}, nil
}
