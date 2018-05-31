package proxy

import (
	"net"
	"time"

	"github.com/juju/errors"
)

type TelegramAddress struct {
	v4 string
	v6 string
}

func (t *TelegramAddress) IPv4() string {
	return net.JoinHostPort(t.v4, telegramPort)
}

func (t *TelegramAddress) IPv6() string {
	return net.JoinHostPort(t.v6, telegramPort)
}

var TelegramAddresses = []TelegramAddress{
	TelegramAddress{v4: "149.154.175.50", v6: "2001:b28:f23d:f001::a"},
	TelegramAddress{v4: "149.154.167.51", v6: "2001:67c:04e8:f002::a"},
	TelegramAddress{v4: "149.154.175.100", v6: "2001:b28:f23d:f003::a"},
	TelegramAddress{v4: "149.154.167.91", v6: "2001:67c:04e8:f004::a"},
	TelegramAddress{v4: "149.154.171.5", v6: "2001:b28:f23f:f005::a"},
}

const telegramPort = "443"

const telegramKeepAlive = 30 * time.Second

func dialToTelegram(ipv6 bool, dcIdx int16, timeout time.Duration) (net.Conn, error) {
	if dcIdx < 0 || dcIdx >= 5 {
		return nil, errors.New("Incorrect DC IDX")
	}

	conn, err := doDial(ipv6, dcIdx, timeout)
	if err != nil {
		return nil, errors.Annotate(err, "Cannot dial")
	}

	if err := conn.SetKeepAlive(true); err != nil {
		return nil, errors.Annotate(err, "Cannot establish keepalive connection")
	}
	if err := conn.SetKeepAlivePeriod(telegramKeepAlive); err != nil {
		return nil, errors.Annotate(err, "Cannot set keepalive timeout")
	}

	return conn, nil
}

func doDial(ipv6 bool, dcIdx int16, timeout time.Duration) (*net.TCPConn, error) {
	dialer := net.Dialer{Timeout: timeout}
	addr := TelegramAddresses[dcIdx]

	if ipv6 {
		if conn, err := dialer.Dial("tcp", addr.IPv6()); err == nil {
			return conn.(*net.TCPConn), nil
		}
	}

	conn, err := dialer.Dial("tcp", addr.IPv4())
	if err == nil {
		return conn.(*net.TCPConn), nil
	}
	return nil, err
}
