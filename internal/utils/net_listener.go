package utils

import (
	"fmt"
	"net"

	"github.com/9seconds/mtg/v2/network"
)

type Listener struct {
	net.Listener
}

// Accept returns the next connection with client socket options applied. A
// connection whose options cannot be set (typically it was reset by the client
// right after the handshake) is closed and skipped: returning an error here
// would stop the whole accept loop because of one bad client.
func (l Listener) Accept() (net.Conn, error) {
	for {
		conn, err := l.Listener.Accept()
		if err != nil {
			return nil, err //nolint: wrapcheck
		}

		if err := network.SetClientSocketOptions(conn, 0); err != nil {
			conn.Close() //nolint: errcheck

			continue
		}

		return conn, nil
	}
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
