package telegram

import (
	"io"
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

type directTelegram struct {
	baseTelegram
}

func (t *directTelegram) Dial(connOpts *mtproto.ConnectionOpts) (io.ReadWriteCloser, error) {
	dc := connOpts.DC
	if dc < 0 {
		dc = -dc
	} else if dc == 0 {
		dc = 1
	}

	return t.baseTelegram.dial(dc-1, connOpts.ConnectionProto)
}

func (t *directTelegram) Init(connOpts *mtproto.ConnectionOpts, conn io.ReadWriteCloser) (io.ReadWriteCloser, error) {
	obfs2, frame := obfuscated2.MakeTelegramObfuscated2Frame(connOpts)
	defer obfuscated2.ReturnFrame(frame)

	if n, err := conn.Write(*frame); err != nil || n != len(*frame) {
		return nil, errors.Annotate(err, "Cannot write hadnshake frame")
	}

	return wrappers.NewStreamCipherRWC(conn, obfs2.Encryptor, obfs2.Decryptor), nil
}

// NewDirectTelegram returns Telegram instance which connects directly
// to Telegram bypassing middleproxies.
func NewDirectTelegram(conf *config.Config) Telegram {
	return &directTelegram{baseTelegram{
		dialer:      tgDialer{net.Dialer{Timeout: telegramDialTimeout}},
		v4Addresses: directV4Addresses,
		v6Addresses: directV6Addresses,
	}}
}
