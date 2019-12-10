package telegram

import (
	"errors"
	"math/rand"
	"net"

	"go.uber.org/zap"

	"github.com/9seconds/mtg/conntypes"
	"github.com/9seconds/mtg/utils"
	"github.com/9seconds/mtg/wrappers/stream"
)

type baseTelegram struct {
	dialer net.Dialer
	logger *zap.SugaredLogger

	secret      []byte
	v4DefaultDC conntypes.DC
	v6DefaultDC conntypes.DC
	v4Addresses map[conntypes.DC][]string
	v6Addresses map[conntypes.DC][]string
}

func (b *baseTelegram) Secret() []byte {
	return b.secret
}

func (b *baseTelegram) dial(dc conntypes.DC,
	protocol conntypes.ConnectionProtocol) (conntypes.StreamReadWriteCloser, error) {
	addresses := make([]string, 0, 2)

	if protocol&conntypes.ConnectionProtocolIPv6 != 0 {
		addresses = append(addresses, b.chooseAddress(b.v6Addresses, dc, b.v6DefaultDC))
	}

	if protocol&conntypes.ConnectionProtocolIPv4 != 0 {
		addresses = append(addresses, b.chooseAddress(b.v4Addresses, dc, b.v4DefaultDC))
	}

	for _, addr := range addresses {
		conn, err := b.dialer.Dial("tcp", addr)
		if err != nil {
			b.logger.Infow("Cannot dial to Telegram", "address", addr, "error", err)
			continue
		}

		if err := utils.InitTCP(conn); err != nil {
			b.logger.Infow("Cannot initialize TCP socket", "address", addr, "error", err)
			continue
		}

		return stream.NewTelegramConn(dc, conn), nil
	}

	return nil, errors.New("cannot dial to the chosen DC")
}

func (b *baseTelegram) chooseAddress(addresses map[conntypes.DC][]string,
	dc, defaultDC conntypes.DC) string {
	addrs, ok := addresses[dc]
	if !ok {
		addrs = addresses[defaultDC]
	}

	switch {
	case len(addrs) == 1:
		return addrs[0]
	case len(addrs) > 1:
		return addrs[rand.Intn(len(addrs))]
	}

	return ""
}
