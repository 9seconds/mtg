package telegram

import (
	"context"
	"errors"
	"time"

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

const (
	defaultDC                    = 2
	defaultUpdateDCAddressesEach = time.Hour
	defaultAppID                 = 123456
	defaultAppHash               = ""
)

type loggerInterface interface {
	Info(msg string)
	WarningError(msg string, err error)
}

type tgAddr struct {
	network string
	address string
}

// https://github.com/telegramdesktop/tdesktop/blob/master/Telegram/SourceFiles/mtproto/mtproto_dc_options.cpp#L30
var (
	defaultV4Addresses = map[int][]tgAddr{
		1: {
			{network: "tcp4", address: "149.154.175.50:443"},
		},
		2: {
			{network: "tcp4", address: "149.154.167.51:443"},
			{network: "tcp4", address: "95.161.76.100:443"},
		},
		3: {
			{network: "tcp4", address: "149.154.175.100:443"},
		},
		4: {
			{network: "tcp4", address: "149.154.167.91:443"},
		},
		5: {
			{network: "tcp4", address: "149.154.171.5:443"},
		},
	}
	defaultV6Addresses = map[int][]tgAddr{
		1: {
			{network: "tcp6", address: "[2001:b28:f23d:f001::a]:443"},
		},
		2: {
			{network: "tcp6", address: "[2001:67c:04e8:f002::a]:443"},
		},
		3: {
			{network: "tcp6", address: "[2001:b28:f23d:f003::a]:443"},
		},
		4: {
			{network: "tcp6", address: "[2001:67c:04e8:f004::a]:443"},
		},
		5: {
			{network: "tcp6", address: "[2001:b28:f23f:f005::a]:443"},
		},
	}
)

type Dialer interface {
	DialContext(ctx context.Context, network, address string) (essentials.Conn, error)
}
