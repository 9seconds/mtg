package telegram

import (
	"io"
	"net"
	"time"

	"github.com/juju/errors"

	"github.com/9seconds/mtg/config"
)

const (
	telegramDialTimeout = 10 * time.Second
)

type tgDialer struct {
	net.Dialer
}

func (t *tgDialer) dial(addr string) (net.Conn, error) {
	conn, err := t.Dialer.Dial("tcp", addr)
	if err != nil {
		return nil, errors.Annotate(err, "Cannot connect to Telegram")
	}
	if err = config.SetSocketOptions(conn); err != nil {
		return nil, errors.Annotate(err, "Cannot set socket options")
	}

	return conn, nil
}

func (t *tgDialer) dialRWC(addr string) (io.ReadWriteCloser, error) {
	conn, err := t.dial(addr)
	if err != nil {
		return nil, err
	}

	return conn, nil
}
