package base

import (
	"math/rand"

	"github.com/juju/errors"

	"github.com/9seconds/mtg/mtproto"
	"github.com/9seconds/mtg/telegram/dialer"
	"github.com/9seconds/mtg/wrappers"
)

type BaseTelegram struct {
	Dialer dialer.Dialer

	V4Addresses map[int16][]string
	V6Addresses map[int16][]string
}

func (b *BaseTelegram) Dial(dcIdx int16, proto mtproto.ConnectionProtocol) (wrappers.StreamReadWriteCloser, error) {
	addrs := make([]string, 2)

	if proto&mtproto.ConnectionProtocolIPv6 != 0 {
		if addr, ok := b.V6Addresses[dcIdx]; ok && len(addr) > 0 {
			addrs = append(addrs, addr[rand.Intn(len(addr))])
		}
	}
	if proto&mtproto.ConnectionProtocolIPv4 != 0 {
		if addr, ok := b.V4Addresses[dcIdx]; ok && len(addr) > 0 {
			addrs = append(addrs, addr[rand.Intn(len(addr))])
		}
	}

	for _, addr := range addrs {
		if conn, err := b.Dialer.Dial(addr); err == nil {
			return conn, err
		}
	}

	return nil, errors.New("Cannot connect to Telegram")
}
