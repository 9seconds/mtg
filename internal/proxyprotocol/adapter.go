package proxyprotocol

import (
	"net"

	"github.com/pires/go-proxyproto"
)

type ListenerAdapter struct {
	proxyproto.Listener
}

func (l *ListenerAdapter) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}

	return connWrapper{conn.(*proxyproto.Conn)}, nil
}
