package telegram

import (
	"math/rand"

	"github.com/juju/errors"

	"github.com/9seconds/mtg/mtproto"
	"github.com/9seconds/mtg/wrappers"
)

type baseTelegram struct {
	dialer tgDialer

	v4Addresses map[int16][]string
	v6Addresses map[int16][]string
}

func (b *baseTelegram) dial(dcIdx int16, connID string, proto mtproto.ConnectionProtocol) (wrappers.WrapStreamReadWriteCloser, error) {
	addrs := make([]string, 2)

	if proto&mtproto.ConnectionProtocolIPv6 != 0 {
		if addr, ok := b.v6Addresses[dcIdx]; ok && len(addr) > 0 {
			addrs = append(addrs, addr[rand.Intn(len(addr))])
		}
	}
	if proto&mtproto.ConnectionProtocolIPv4 != 0 {
		if addr, ok := b.v4Addresses[dcIdx]; ok && len(addr) > 0 {
			addrs = append(addrs, addr[rand.Intn(len(addr))])
		}
	}

	for _, addr := range addrs {
		if conn, err := b.dialer.dialRWC(addr, connID); err == nil {
			return conn, err
		}
	}

	return nil, errors.New("Cannot connect to Telegram")
}
