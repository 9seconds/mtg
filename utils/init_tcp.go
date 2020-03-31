package utils

import (
	"fmt"
	"net"
	"time"
)

const tcpKeepAlivePingPeriod = 2 * time.Second

func InitTCP(conn net.Conn, readBufferSize int, writeBufferSize int) error {
	tcpConn := conn.(*net.TCPConn)

	if err := tcpConn.SetNoDelay(true); err != nil {
		return fmt.Errorf("cannot set TCP_NO_DELAY: %w", err)
	}

	if err := tcpConn.SetReadBuffer(readBufferSize); err != nil {
		return fmt.Errorf("cannot set read buffer size: %w", err)
	}

	if err := tcpConn.SetWriteBuffer(writeBufferSize); err != nil {
		return fmt.Errorf("cannot set write buffer size: %w", err)
	}

	if err := tcpConn.SetKeepAlive(true); err != nil {
		return fmt.Errorf("cannot enable keep-alive: %w", err)
	}

	if err := tcpConn.SetKeepAlivePeriod(tcpKeepAlivePingPeriod); err != nil {
		return fmt.Errorf("cannot set keep-alive period: %w", err)
	}

	return nil
}
