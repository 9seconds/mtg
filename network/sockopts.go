package network

import (
	"fmt"
	"net"
)

// SetClientSocketOptions tunes a TCP socket that represents a connection to
// end user (not Telegram service or fronting domain).
//
// bufferSize setting is deprecated and ignored.
func SetClientSocketOptions(conn net.Conn, bufferSize int) error {
	return setCommonSocketOptions(conn.(*net.TCPConn)) //nolint: forcetypeassert
}

// SetServerSocketOptions tunes a TCP socket that represents a connection to
// remote server like Telegram or fronting domain (but not end user).
func SetServerSocketOptions(conn net.Conn, bufferSize int) error {
	return setCommonSocketOptions(conn.(*net.TCPConn)) //nolint: forcetypeassert
}

func setCommonSocketOptions(conn *net.TCPConn) error {
	if err := conn.SetKeepAlivePeriod(DefaultTCPKeepAlivePeriod); err != nil {
		return fmt.Errorf("cannot set time period of TCP keepalive probes: %w", err)
	}

	if err := conn.SetLinger(tcpLingerTimeout); err != nil {
		return fmt.Errorf("cannot set TCP linger timeout: %w", err)
	}

	rawConn, err := conn.SyscallConn()
	if err != nil {
		return fmt.Errorf("cannot get underlying raw connection: %w", err)
	}

	if err := setSocketReuseAddrPort(rawConn); err != nil {
		return fmt.Errorf("cannot setup SO_REUSEADDR/PORT: %w", err)
	}

	return nil
}
