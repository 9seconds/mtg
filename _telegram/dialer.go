package telegram

import (
	"context"
	"net"
	"time"

	"github.com/juju/errors"

	"github.com/9seconds/mtg/config"
	"github.com/9seconds/mtg/wrappers"
)

const telegramDialTimeout = 10 * time.Second

type tgDialer struct {
	net.Dialer

	conf *config.Config
}

func (t *tgDialer) dial(addr string) (net.Conn, error) {
	conn, err := t.Dialer.Dial("tcp", addr)
	if err != nil {
		return nil, errors.Annotate(err, "Cannot connect to Telegram")
	}

	tcpSocket := conn.(*net.TCPConn)
	if err = tcpSocket.SetNoDelay(true); err != nil {
		return nil, errors.Annotate(err, "Cannot set NO_DELAY to Telegram")
	}
	if err = tcpSocket.SetReadBuffer(t.conf.WriteBufferSize); err != nil {
		return nil, errors.Annotate(err, "Cannot set read buffer size on telegram socket")
	}
	if err = tcpSocket.SetWriteBuffer(t.conf.ReadBufferSize); err != nil {
		return nil, errors.Annotate(err, "Cannot set write buffer size on telegram socket")
	}

	return conn, nil
}

func (t *tgDialer) dialRWC(ctx context.Context, cancel context.CancelFunc,
	addr, connID string) (wrappers.StreamReadWriteCloser, error) {
	conn, err := t.dial(addr)
	if err != nil {
		return nil, err
	}
	tgConn := wrappers.NewConn(ctx, cancel, conn, connID,
		wrappers.ConnPurposeTelegram, t.conf.PublicIPv4, t.conf.PublicIPv6)

	return tgConn, nil
}
