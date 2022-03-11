package telegram

import (
	"context"
	"errors"

	"github.com/9seconds/mtg/v2/essentials"
)

var errNoAddresses = errors.New("no addresses")

type preferIP uint8

const (
	preferIPOnlyIPv4 preferIP = iota
	preferIPOnlyIPv6
	preferIPPreferIPv4
	preferIPPreferIPv6
)

type tgAddr struct {
	network string
	address string
}

// https://github.com/telegramdesktop/tdesktop/blob/master/Telegram/SourceFiles/mtproto/mtproto_dc_options.cpp#L30
var (
	productionV4Addresses = [][]tgAddr{
		{ // dc1
			{network: "tcp4", address: "149.154.175.50:443"},
		},
		{ // dc2
			{network: "tcp4", address: "149.154.167.51:443"},
			{network: "tcp4", address: "95.161.76.100:443"},
		},
		{ // dc3
			{network: "tcp4", address: "149.154.175.100:443"},
		},
		{ // dc4
			{network: "tcp4", address: "149.154.167.91:443"},
		},
		{ // dc5
			{network: "tcp4", address: "149.154.171.5:443"},
		},
	}
	productionV6Addresses = [][]tgAddr{
		{ // dc1
			{network: "tcp6", address: "[2001:b28:f23d:f001::a]:443"},
		},
		{ // dc2
			{network: "tcp6", address: "[2001:67c:04e8:f002::a]:443"},
		},
		{ // dc3
			{network: "tcp6", address: "[2001:b28:f23d:f003::a]:443"},
		},
		{ // dc4
			{network: "tcp6", address: "[2001:67c:04e8:f004::a]:443"},
		},
		{ // dc5
			{network: "tcp6", address: "[2001:b28:f23f:f005::a]:443"},
		},
	}

	testV4Addresses = [][]tgAddr{
		{ // dc1
			{network: "tcp4", address: "149.154.175.10:443"},
		},
		{ // dc2
			{network: "tcp4", address: "149.154.167.40:443"},
		},
		{ // dc3
			{network: "tcp4", address: "149.154.175.117:443"},
		},
	}
	testV6Addresses = [][]tgAddr{
		{ // dc1
			{network: "tcp6", address: "[2001:b28:f23d:f001::e]:443"},
		},
		{ // dc2
			{network: "tcp6", address: "[2001:67c:04e8:f002::e]:443"},
		},
		{ // dc3
			{network: "tcp6", address: "[2001:b28:f23d:f003::e]:443"},
		},
	}
)

type Dialer interface {
	DialContext(ctx context.Context, network, address string) (essentials.Conn, error)
}
