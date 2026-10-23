package utils

import (
	"context"
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
		conn.Close() //nolint: errcheck

		return nil, fmt.Errorf("cannot set TCP options: %w", err)
	}

	return conn, nil
}

// NewListener opens a listening socket. listenerMSS > 0 sets TCP_MAXSEG on it
// (Linux only): this MSS is advertised to the client in the SYN-ACK and caps
// the size of segments sent to the client for the whole connection.
func NewListener(bindTo string, bufferSize int, listenerMSS int) (net.Listener, error) {
	lc := net.ListenConfig{Control: network.ListenControlMSS(listenerMSS)}

	base, err := lc.Listen(context.Background(), "tcp", bindTo)
	if err != nil {
		return nil, fmt.Errorf("cannot build a base listener: %w", err)
	}

	return Listener{
		Listener: base,
	}, nil
}
