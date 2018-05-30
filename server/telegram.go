package server

import (
	"net"
	"time"

	"github.com/juju/errors"
)

var telegramDCIPs = [5]string{
	"149.154.175.50:443",
	"149.154.167.51:443",
	"149.154.175.100:443",
	"149.154.167.91:443",
	"149.154.171.5:443",
}

const telegramKeepAlive = 30 * time.Second

func dialToTelegram(dcIdx int16) (net.Conn, error) {
	if dcIdx < 0 || dcIdx >= 5 {
		return nil, errors.New("Incorrect DC IDX")
	}

	tcpAddr, _ := net.ResolveTCPAddr("tcp", telegramDCIPs[dcIdx])
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
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
