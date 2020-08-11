package telegram

import (
	"errors"
	"math/rand"
	"net"

	"github.com/9seconds/mtg/config"
	"github.com/9seconds/mtg/conntypes"
	"github.com/9seconds/mtg/utils"
	"github.com/9seconds/mtg/wrappers/stream"
	"go.uber.org/zap"
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
	for _, addr := range b.getAddresses(dc, protocol) {
		conn, err := b.dialer.Dial("tcp", addr)
		if err != nil {
			b.logger.Infow("Cannot dial to Telegram", "address", addr, "error", err)

			continue
		}

		if err := utils.InitTCP(conn, config.C.ProxyReadBuffer(), config.C.ProxyWriteBuffer()); err != nil {
			b.logger.Infow("Cannot initialize TCP socket", "address", addr, "error", err)

			continue
		}

		return stream.NewTelegramConn(dc, conn), nil
	}

	return nil, errors.New("cannot dial to the chosen DC")
}

func (b *baseTelegram) getAddresses(dc conntypes.DC, protocol conntypes.ConnectionProtocol) []string {
	addresses := make([]string, 0, 2)
	protos := []conntypes.ConnectionProtocol{
		conntypes.ConnectionProtocolIPv6,
		conntypes.ConnectionProtocolIPv4,
	}

	if config.C.PreferIP == config.PreferIPv4 {
		protos[0], protos[1] = protos[1], protos[0]
	}

	for _, proto := range protos {
		switch {
		case proto&protocol == 0:
		case proto&conntypes.ConnectionProtocolIPv6 != 0:
			addresses = append(addresses, b.chooseAddress(b.v6Addresses, dc, b.v6DefaultDC))
		case proto&conntypes.ConnectionProtocolIPv4 != 0:
			addresses = append(addresses, b.chooseAddress(b.v4Addresses, dc, b.v4DefaultDC))
		}
	}

	return addresses
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
		return addrs[rand.Intn(len(addrs))] // nolint: gosec
	}

	return ""
}
