package client

import (
	"net"

	"github.com/9seconds/mtg/config"
	"github.com/9seconds/mtg/mtproto"
	"github.com/9seconds/mtg/wrappers"
)

// MiddleInit initializes client connection for proxy which has to
// support promoted channels, connect to Telegram middle proxies etc.
func MiddleInit(socket net.Conn, conf *config.Config) (wrappers.Wrap, *mtproto.ConnectionOpts, error) {
	conn, opts, err := DirectInit(socket, conf)
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
