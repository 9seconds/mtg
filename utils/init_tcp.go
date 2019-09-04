package utils

import (
	"net"

	"github.com/juju/errors"

	"github.com/9seconds/mtg/config"
)

func InitTCP(conn net.Conn) error {
	tcpConn := conn.(*net.TCPConn)

	if err := tcpConn.SetNoDelay(true); err != nil {
		return errors.Annotate(err, "Cannot set NO_DELAY")
	}
	if err := tcpConn.SetReadBuffer(config.C.BufferSize.Read); err != nil {
		return errors.Annotate(err, "Cannot set read buffer size")
	}
	if err := tcpConn.SetWriteBuffer(config.C.BufferSize.Write); err != nil {
		return errors.Annotate(err, "Cannot set write buffer size")
	}

	return nil
}
