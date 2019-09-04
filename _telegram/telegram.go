package telegram

import (
	"context"
	"math/rand"

	"github.com/juju/errors"

	"github.com/9seconds/mtg/mtproto"
	"github.com/9seconds/mtg/wrappers"
)

// Telegram is an interface for different Telegram work modes.
type Telegram interface {
	Dial(context.Context, context.CancelFunc, string, *mtproto.ConnectionOpts) (wrappers.StreamReadWriteCloser, error)
	Init(*mtproto.ConnectionOpts, wrappers.StreamReadWriteCloser) (wrappers.Wrap, error)
}

type baseTelegram struct {
	dialer tgDialer

	v4DefaultIdx int16
	v6DefaultIdx int16
	v4Addresses  map[int16][]string
	v6Addresses  map[int16][]string
}

func (b *baseTelegram) dial(ctx context.Context, cancel context.CancelFunc, dcIdx int16, connID string,
	proto mtproto.ConnectionProtocol) (wrappers.StreamReadWriteCloser, error) {
	addrs := make([]string, 2)

	if proto&mtproto.ConnectionProtocolIPv6 != 0 {
		if addr := b.chooseAddress(b.v6Addresses, dcIdx, b.v6DefaultIdx); addr != "" {
			addrs = append(addrs, addr)
		}
	}
	if proto&mtproto.ConnectionProtocolIPv4 != 0 {
		if addr := b.chooseAddress(b.v4Addresses, dcIdx, b.v4DefaultIdx); addr != "" {
			addrs = append(addrs, addr)
		}
	}

	for _, addr := range addrs {
		if conn, err := b.dialer.dialRWC(ctx, cancel, addr, connID); err == nil {
			return conn, err
		}
	}

	return nil, errors.New("Cannot connect to Telegram")
}

func (b *baseTelegram) chooseAddress(addresses map[int16][]string, idx, defaultIdx int16) string {
	if addr, ok := addresses[idx]; ok {
		return b.chooseRandomAddress(addr)
	} else if addr, ok := addresses[defaultIdx]; ok {
		return b.chooseRandomAddress(addr)
	}

	return ""
}

func (b *baseTelegram) chooseRandomAddress(addresses []string) string {
	if len(addresses) > 0 {
		return addresses[rand.Intn(len(addresses))]
	}

	return ""
}
