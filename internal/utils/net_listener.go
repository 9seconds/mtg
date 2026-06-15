package utils

import (
	"context"
	"fmt"
	"net"
	"syscall"

	"github.com/9seconds/mtg/v2/essentials"
	"github.com/9seconds/mtg/v2/network"
)

type Listener struct {
	net.Listener
}

func (l Listener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err //nolint: wrapcheck
	}

	if err := network.SetClientSocketOptions(conn, 0); err != nil {
		conn.Close() //nolint: errcheck

		return nil, fmt.Errorf("cannot set TCP options: %w", err)
	}

	return conn, nil
}

func NewListener(bindTo string, windowClamp int) (net.Listener, error) {
	var control func(string, string, syscall.RawConn) error
	if windowClamp > 0 {
		control = func(_, _ string, conn syscall.RawConn) error {
			return essentials.SetRawTCPWindowClamp(conn, windowClamp)
		}
	}

	listenConfig := net.ListenConfig{
		Control: control,
	}

	base, err := listenConfig.Listen(context.Background(), "tcp", bindTo)
	if err != nil {
		return nil, fmt.Errorf("cannot build a base listener: %w", err)
	}

	return Listener{Listener: base}, nil
}
