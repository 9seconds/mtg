package dc

import (
	"context"
	"net"
	"time"

	"github.com/dolonet/mtg-multi/essentials"
)

type preferIP uint8

const (
	preferIPOnlyIPv4 preferIP = iota
	preferIPOnlyIPv6
	preferIPPreferIPv4
	preferIPPreferIPv6
)

const (
	// Default DC to connect to if not sure.
	DefaultDC = 2

	// How often should we request updates from
	// https://core.telegram.org/getProxyConfig
	PublicConfigUpdateEach  = time.Hour
	PublicConfigUpdateURLv4 = "https://core.telegram.org/getProxyConfig"
	PublicConfigUpdateURLv6 = "https://core.telegram.org/getProxyConfigV6"

	// How often should we extract hosts from Telegram using help.getConfig
	// method.
	OwnConfigUpdateEach = time.Hour
)

type Logger interface {
	Info(msg string)
	WarningError(msg string, err error)
}

type Updater interface {
	Run(ctx context.Context)
}

// https://github.com/telegramdesktop/tdesktop/blob/master/Telegram/SourceFiles/mtproto/mtproto_dc_options.cpp#L30
var defaultDCAddrSet = (func() dcAddrSet {
	addrSet := dcAddrSet{
		v4: make(map[int][]Addr),
		v6: make(map[int][]Addr),
	}

	for dcid, ips := range essentials.TelegramCoreAddresses {
		for _, addr := range ips {
			host, _, err := net.SplitHostPort(addr)
			if err != nil {
				panic(err)
			}

			ip := net.ParseIP(host)
			if ip == nil {
				panic(addr)
			}
			if ip.To4() == nil {
				addrSet.v6[dcid] = append(addrSet.v6[dcid], Addr{
					Network: "tcp6",
					Address: addr,
				})
			} else {
				addrSet.v4[dcid] = append(addrSet.v4[dcid], Addr{
					Network: "tcp4",
					Address: addr,
				})
			}
		}
	}

	return addrSet
})()
