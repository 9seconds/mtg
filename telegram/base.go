package telegram

import (
	"fmt"
	"math/rand"
	"net"
	"time"

	"github.com/9seconds/mtg/conntypes"
	"github.com/9seconds/mtg/utils"
	"github.com/9seconds/mtg/wrappers"
)

const telegramDialTimeout = 10 * time.Second

type baseTelegram struct {
	dialer net.Dialer

	secret      []byte
	v4DefaultDC conntypes.DC
	V6DefaultDC conntypes.DC
	v4Addresses map[conntypes.DC][]string
	v6Addresses map[conntypes.DC][]string
}

func (b *baseTelegram) Secret() []byte {
	return b.secret
}

func (b *baseTelegram) dial(dc conntypes.DC,
	protocol conntypes.ConnectionProtocol) (conntypes.StreamReadWriteCloser, error) {
	addr := ""

	switch protocol {
	case conntypes.ConnectionProtocolIPv4:
		addr = b.chooseAddress(b.v4Addresses, dc, b.v4DefaultDC)
	default:
		addr = b.chooseAddress(b.v6Addresses, dc, b.V6DefaultDC)
	}

	conn, err := b.dialer.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("dial has failed: %w", err)
	}

	if err := utils.InitTCP(conn); err != nil {
		return nil, fmt.Errorf("cannot initialize tcp socket: %w", err)
	}

	return wrappers.NewTelegramConn(conn), nil
}

func (b *baseTelegram) chooseAddress(addresses map[conntypes.DC][]string,
	dc, defaultDC conntypes.DC) string {
	addrs, ok := addresses[dc]
	if !ok {
		addrs, _ = addresses[defaultDC]
	}

	switch {
	case len(addrs) == 1:
		return addrs[0]
	case len(addrs) > 1:
		return addrs[rand.Intn(len(addrs))]
	}

	return ""
}
