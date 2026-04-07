package network

import (
	"fmt"
	"net"
)

func setCommonSocketOptions(conn *net.TCPConn) error {
	if err := conn.SetKeepAliveConfig(net.KeepAliveConfig{
		Enable:   true,
		Idle:     DefaultKeepAliveIdle,
		Interval: DefaultKeepAliveInterval,
		Count:    DefaultKeepAliveCount,
	}); err != nil {
		return fmt.Errorf("cannot configure TCP keepalive: %w", err)
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

	setCongestionControl(rawConn)

	return nil
}
