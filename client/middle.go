package client

import (
	"net"

	"github.com/9seconds/mtg/config"
	"github.com/9seconds/mtg/mtproto"
	mtwrappers "github.com/9seconds/mtg/mtproto/wrappers"
	"github.com/9seconds/mtg/wrappers"
)

func MiddleInit(conn net.Conn, conf *config.Config) (*mtproto.ConnectionOpts, wrappers.ReadWriteCloserWithAddr, error) {
	opts, newConn, err := DirectInit(conn, conf)
	if err != nil {
		return nil, nil, err
	}

	if opts.ConnectionType == mtproto.ConnectionTypeAbridged {
		newConn = mtwrappers.NewAbridgedRWC(newConn, opts)
	} else {
		newConn = mtwrappers.NewIntermediateRWC(newConn, opts)
	}

	opts.ConnectionProto = mtproto.ConnectionProtocolIPv4
	if conn.LocalAddr().(*net.TCPAddr).IP.To4() == nil {
		opts.ConnectionProto = mtproto.ConnectionProtocolIPv6
	}

	return opts, newConn, nil
}
