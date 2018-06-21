package telegram

import (
	"io"
	"net"
	"time"

	"github.com/juju/errors"

	"github.com/9seconds/mtg/config"
	"github.com/9seconds/mtg/wrappers"
)

const telegramKeepAlive = 30 * time.Second

type tgDialer struct {
	net.Dialer

	conf *config.Config
}

func (t *tgDialer) dial(addr string) (net.Conn, error) {
	connRaw, err := t.Dialer.Dial("tcp", addr)
	if err != nil {
		return nil, errors.Annotate(err, "Cannot connect to Telegram")
	}
	conn := connRaw.(*net.TCPConn)

	if err = conn.SetKeepAlive(true); err != nil {
		return nil, errors.Annotate(err, "Cannot establish keepalive connection")
	}
	if err = conn.SetKeepAlivePeriod(telegramKeepAlive); err != nil {
		return nil, errors.Annotate(err, "Cannot set keepalive timeout")
	}

	return conn, nil
}

func (t *tgDialer) dialRWC(addr string) (io.ReadWriteCloser, error) {
	conn, err := t.dial(addr)
	if err != nil {
		return nil, err
	}

	return wrappers.NewTimeoutRWC(conn, t.conf.TimeoutRead, t.conf.TimeoutWrite), nil
}

func newDialer(conf *config.Config) tgDialer {
	return tgDialer{
		Dialer: net.Dialer{Timeout: conf.TimeoutRead},
		conf:   conf,
	}
}
