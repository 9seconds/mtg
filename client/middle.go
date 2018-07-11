package client

import (
	"net"

	"mtg/config"
	"mtg/mtproto"
	"mtg/wrappers"
)

// MiddleInit initializes client connection for proxy which has to
// support promoted channels, connect to Telegram middle proxies etc.
func MiddleInit(socket net.Conn, connID string, conf *config.Config) (wrappers.Wrap, *mtproto.ConnectionOpts, error) {
	conn, opts, err := DirectInit(socket, connID, conf)
	if err != nil {
		return nil, nil, err
	}
	connStream := conn.(wrappers.StreamReadWriteCloser)

	newConn := wrappers.NewMTProtoAbridged(connStream, opts)
	if opts.ConnectionType != mtproto.ConnectionTypeAbridged {
		newConn = wrappers.NewMTProtoIntermediate(connStream, opts)
	}

	opts.ConnectionProto = mtproto.ConnectionProtocolIPv4
	if socket.LocalAddr().(*net.TCPAddr).IP.To4() == nil {
		opts.ConnectionProto = mtproto.ConnectionProtocolIPv6
	}

	return newConn, opts, err
}
