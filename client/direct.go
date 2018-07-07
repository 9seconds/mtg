package client

import (
	"context"
	"net"
	"time"

	"github.com/juju/errors"

	"github.com/9seconds/mtg/config"
	"github.com/9seconds/mtg/mtproto"
	"github.com/9seconds/mtg/obfuscated2"
	"github.com/9seconds/mtg/wrappers"
)

const handshakeTimeout = 10 * time.Second

func DirectInit(ctx context.Context, cancel context.CancelFunc, socket net.Conn, connID string,
	conf *config.Config) (wrappers.WrapStreamReadWriteCloser, *mtproto.ConnectionOpts, error) {
	if err := config.SetSocketOptions(socket); err != nil {
		return nil, nil, errors.Annotate(err, "Cannot set socket options")
	}

	socket.SetReadDeadline(time.Now().Add(handshakeTimeout))
	frame, err := obfuscated2.ExtractFrame(socket)
	if err != nil {
		return nil, nil, errors.Annotate(err, "Cannot extract frame")
	}
	socket.SetReadDeadline(time.Time{})
	conn := wrappers.NewConn(socket, connID, wrappers.ConnPurposeClient, conf.PublicIPv4, conf.PublicIPv6)

	obfs2, connOpts, err := obfuscated2.ParseObfuscated2ClientFrame(conf.Secret, frame)
	if err != nil {
		return nil, nil, errors.Annotate(err, "Cannot parse obfuscated frame")
	}
	connOpts.ConnectionProto = mtproto.ConnectionProtocolAny
	connOpts.ClientAddr = conn.RemoteAddr()

	conn = wrappers.NewCtx(ctx, cancel, conn)
	conn = wrappers.NewStreamCipher(conn, obfs2.Encryptor, obfs2.Decryptor)

	return conn, connOpts, nil
}
