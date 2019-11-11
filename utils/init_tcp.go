package utils

import (
	"fmt"
	"net"

	"mtg/config"
)

func InitTCP(conn net.Conn) error {
	tcpConn := conn.(*net.TCPConn)

	if err := tcpConn.SetNoDelay(true); err != nil {
		return fmt.Errorf("cannot set TCP_NO_DELAY: %w", err)
	}

	if err := tcpConn.SetReadBuffer(config.C.ReadBuffer); err != nil {
		return fmt.Errorf("cannot set read buffer size: %w", err)
	}

	if err := tcpConn.SetWriteBuffer(config.C.WriteBuffer); err != nil {
		return fmt.Errorf("cannot set write buffer size: %w", err)
	}

	return nil
}
