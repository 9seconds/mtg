package proxyprotocol

import "github.com/pires/go-proxyproto"

type connWrapper struct {
	*proxyproto.Conn
}

func (c connWrapper) CloseRead() error {
	tcpConn, ok := c.TCPConn()
	if !ok {
		panic("we support only tcp connections")
	}

	return tcpConn.CloseRead()
}

func (c connWrapper) CloseWrite() error {
	tcpConn, ok := c.TCPConn()
	if !ok {
		panic("we support only tcp connections")
	}

	return tcpConn.CloseWrite()
}
