package telegram

import (
	"net"
	"time"

	"github.com/juju/errors"

	"github.com/9seconds/mtg/config"
	"github.com/9seconds/mtg/wrappers"
)

const (
	telegramDialTimeout = 10 * time.Second
	readBufferSize      = 64 * 1024
	writeBufferSize     = 64 * 1024
)

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
	if err = tcpSocket.SetReadBuffer(readBufferSize); err != nil {
		return nil, errors.Annotate(err, "Cannot set read buffer size on telegram socket")
	}
	if err = tcpSocket.SetWriteBuffer(writeBufferSize); err != nil {
		return nil, errors.Annotate(err, "Cannot set write buffer size on telegram socket")
	}

	return conn, nil
}

func (t *tgDialer) dialRWC(addr, connID string) (wrappers.StreamReadWriteCloser, error) {
	conn, err := t.dial(addr)
	if err != nil {
		return nil, err
	}
	tgConn := wrappers.NewConn(conn, connID, wrappers.ConnPurposeTelegram, t.conf.PublicIPv4, t.conf.PublicIPv6)

	return tgConn, nil
}
