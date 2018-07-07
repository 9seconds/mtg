package client

import (
	"context"
	"net"

	"github.com/9seconds/mtg/config"
	"github.com/9seconds/mtg/mtproto"
	"github.com/9seconds/mtg/wrappers"
)

func MiddleInit(ctx context.Context, cancel context.CancelFunc, socket net.Conn, connID string,
	conf *config.Config) (wrappers.WrapPacketReadWriteCloser, *mtproto.ConnectionOpts, error) {
	conn, opts, err := DirectInit(ctx, cancel, socket, connID, conf)
	if err != nil {
		return nil, nil, err
	}

	newConn := wrappers.NewMTProtoAbridged(conn, opts)
	if opts.ConnectionType != mtproto.ConnectionTypeAbridged {
		newConn = wrappers.NewMTProtoIntermediate(conn, opts)
	}

	opts.ConnectionProto = mtproto.ConnectionProtocolIPv4
	if socket.LocalAddr().(*net.TCPAddr).IP.To4() == nil {
		opts.ConnectionProto = mtproto.ConnectionProtocolIPv6
	}

	return newConn, opts, err
}
