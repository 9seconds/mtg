package utils

import (
	"fmt"
	"net"
	"strings"

	"github.com/9seconds/mtg/v2/network"
	sam "github.com/eyedeekay/sam3/helper"
)

type Listener struct {
	net.Listener
}

func (l Listener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err // nolint: wrapcheck
	}

	if err := network.SetClientSocketOptions(conn, 0); err != nil {
		conn.Close()

		return nil, fmt.Errorf("cannot set TCP options: %w", err)
	}

	return conn, nil
}

func NewListener(bindTo string, bufferSize int) (net.Listener, error) {
	if strings.HasSuffix(bindTo, ".i2p") {
		base, err := sam.I2PListener(bindTo, "127.0.0.1:7656", bindTo)
		if err != nil {
			return nil, fmt.Errorf("cannot build a base I2P listener: %w", err)
		}
		return Listener{
			Listener: base,
		}, nil
	}
	base, err := net.Listen("tcp", bindTo)
	if err != nil {
		return nil, fmt.Errorf("cannot build a base listener: %w", err)
	}

	return Listener{
		Listener: base,
	}, nil
}
