package telegram

import (
	"net"

	"github.com/juju/errors"

	"github.com/9seconds/mtg/config"
	"github.com/9seconds/mtg/mtproto"
	"github.com/9seconds/mtg/obfuscated2"
	"github.com/9seconds/mtg/wrappers"
)

var (
	directV4Addresses = map[int16][]string{
		0: []string{"149.154.175.50:443"},
		1: []string{"149.154.167.51:443"},
		2: []string{"149.154.175.100:443"},
		3: []string{"149.154.167.91:443"},
		4: []string{"149.154.171.5:443"},
	}
	directV6Addresses = map[int16][]string{
		0: []string{"[2001:b28:f23d:f001::a]:443"},
		1: []string{"[2001:67c:04e8:f002::a]:443"},
		2: []string{"[2001:b28:f23d:f003::a]:443"},
		3: []string{"[2001:67c:04e8:f004::a]:443"},
		4: []string{"[2001:b28:f23f:f005::a]:443"},
	}
)

type DirectTelegram struct {
	baseTelegram
}

func (t *DirectTelegram) Dial(connID string, connOpts *mtproto.ConnectionOpts) (wrappers.StreamReadWriteCloser, error) {
	dc := connOpts.DC
	if dc < 0 {
		dc = -dc
	} else if dc == 0 {
		dc = 1
	}

	return t.baseTelegram.dial(dc-1, connID, connOpts.ConnectionProto)
}

func (t *DirectTelegram) Init(connOpts *mtproto.ConnectionOpts, conn wrappers.StreamReadWriteCloser) (wrappers.Wrap, error) {
	obfs2, frame := obfuscated2.MakeTelegramObfuscated2Frame(connOpts)

	if _, err := conn.Write(frame); err != nil {
		return nil, errors.Annotate(err, "Cannot write hadnshake frame")
	}

	return wrappers.NewStreamCipher(conn, obfs2.Encryptor, obfs2.Decryptor), nil
}

// NewDirectTelegram returns Telegram instance which connects directly
// to Telegram bypassing middleproxies.
func NewDirectTelegram(conf *config.Config) Telegram {
	return &DirectTelegram{baseTelegram{
		dialer: tgDialer{
			Dialer: net.Dialer{Timeout: telegramDialTimeout},
			conf:   conf,
		},
		v4Addresses: directV4Addresses,
		v6Addresses: directV6Addresses,
	}}
}
