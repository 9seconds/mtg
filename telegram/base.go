package telegram

import (
	"context"
	"math/rand"
	"net"
	"time"

	"github.com/juju/errors"

	"github.com/9seconds/mtg/conntypes"
	"github.com/9seconds/mtg/utils"
	"github.com/9seconds/mtg/wrappers"
)

const telegramDialTimeout = 10 * time.Second

type baseTelegram struct {
	dialer net.Dialer

	v4DefaultDC conntypes.DC
	V6DefaultDC conntypes.DC
	v4Addresses map[conntypes.DC][]string
	v6Addresses map[conntypes.DC][]string
}

func (b *baseTelegram) dialToAddress(ctx context.Context,
	cancel context.CancelFunc,
	addr string) (wrappers.StreamReadWriteCloser, error) {
	conn, err := b.dialer.Dial("tcp", addr)
	if err != nil {
		return nil, errors.Annotate(err, "Dial has failed")
	}

	if err := utils.InitTCP(conn); err != nil {
		return nil, errors.Annotate(err, "Cannot initialize TCP socket")
	}

	return wrappers.NewTelegramConn(ctx, cancel, conn), nil
}

func (b *baseTelegram) dial(ctx context.Context,
	cancel context.CancelFunc,
	dc conntypes.DC,
	protocol conntypes.ConnectionProtocol) (wrappers.StreamReadWriteCloser, error) {
	addr := ""

	switch protocol {
	case conntypes.ConnectionProtocolIPv4:
		addr = b.chooseAddress(b.v4Addresses, dc, b.v4DefaultDC)
	default:
		addr = b.chooseAddress(b.v6Addresses, dc, b.V6DefaultDC)
	}

	return b.dialToAddress(ctx, cancel, addr)
}

func (b *baseTelegram) chooseAddress(addresses map[conntypes.DC][]string,
	dc, defaultDC conntypes.DC) string {
	addrs, ok := addresses[dc]
	if !ok {
		addrs, _ = addresses[defaultDC]
	}

	if len(addrs) > 0 {
		return addrs[rand.Intn(len(addrs))]
	}

	return ""
}
