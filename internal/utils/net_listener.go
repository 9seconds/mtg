package utils

import (
	"fmt"
	"net"

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
		conn.Close()

		return nil, fmt.Errorf("cannot set TCP options: %w", err)
	}

	return conn, nil
}

func NewListener(bindTo string, bufferSize int) (net.Listener, error) {
	base, err := net.Listen("tcp", bindTo)
	if err != nil {
		return nil, fmt.Errorf("cannot build a base listener: %w", err)
	}

	return Listener{
		Listener: base,
	}, nil
}
