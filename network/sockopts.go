package network

import (
	"fmt"
	"net"

	"golang.org/x/sys/unix"
)

// SetClientSocketOptions tunes a TCP socket that represents a connection to
// end user (not Telegram service or fronting domain).
func SetClientSocketOptions(conn net.Conn, bufferSize int) error {
	tcpConn := conn.(*net.TCPConn) // nolint: forcetypeassert

	if err := tcpConn.SetNoDelay(false); err != nil {
		return fmt.Errorf("cannot disable TCP_NO_DELAY: %w", err)
	}

	return setCommonSocketOptions(tcpConn, bufferSize)
}

// SetServerSocketOptions tunes a TCP socket that represents a connection to
// remote server like Telegram or fronting domain (but not end user).
func SetServerSocketOptions(conn net.Conn, bufferSize int) error {
	tcpConn := conn.(*net.TCPConn) // nolint: forcetypeassert

	if err := tcpConn.SetNoDelay(true); err != nil {
		return fmt.Errorf("cannot enable TCP_NO_DELAY: %w", err)
	}

	return setCommonSocketOptions(tcpConn, bufferSize)
}

func setCommonSocketOptions(conn *net.TCPConn, bufferSize int) error {
	if err := conn.SetReadBuffer(bufferSize); err != nil {
		return fmt.Errorf("cannot set read buffer size: %w", err)
	}

	if err := conn.SetWriteBuffer(bufferSize); err != nil {
		return fmt.Errorf("cannot set write buffer size: %w", err)
	}

	if err := conn.SetKeepAlive(false); err != nil {
		return fmt.Errorf("cannot disable TCP keepalive probes: %w", err)
	}

	if err := conn.SetLinger(tcpLingerTimeout); err != nil {
		return fmt.Errorf("cannot set TCP linger timeout: %w", err)
	}

	rawConn, err := conn.SyscallConn()
	if err != nil {
		return fmt.Errorf("cannot get underlying raw connection: %w", err)
	}

	rawConn.Control(func(fd uintptr) { // nolint: errcheck
		err = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEADDR, 1)
		if err != nil {
			err = fmt.Errorf("cannot set SO_REUSEADDR: %w", err)

			return
		}

		err = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEPORT, 1)
		if err != nil {
			err = fmt.Errorf("cannot set SO_REUSEPORT: %w", err)
		}
	})

	return nil
}
