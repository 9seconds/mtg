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

// DirectInit initializes client connection for proxy which connects to
// Telegram directly.
func DirectInit(ctx context.Context, cancel context.CancelFunc, socket net.Conn,
	connID string, conf *config.Config) (wrappers.Wrap, *mtproto.ConnectionOpts, error) {
	tcpSocket := socket.(*net.TCPConn)
	if err := tcpSocket.SetNoDelay(false); err != nil {
		return nil, nil, errors.Annotate(err, "Cannot disable NO_DELAY to client socket")
	}
	if err := tcpSocket.SetReadBuffer(conf.ReadBufferSize); err != nil {
		return nil, nil, errors.Annotate(err, "Cannot set read buffer size of client socket")
	}
	if err := tcpSocket.SetWriteBuffer(conf.WriteBufferSize); err != nil {
		return nil, nil, errors.Annotate(err, "Cannot set write buffer size of client socket")
	}

	socket.SetReadDeadline(time.Now().Add(handshakeTimeout)) // nolint: errcheck
	frame, err := obfuscated2.ExtractFrame(socket)
	if err != nil {
		return nil, nil, errors.Annotate(err, "Cannot extract frame")
	}
	socket.SetReadDeadline(time.Time{}) // nolint: errcheck

	conn := wrappers.NewConn(ctx, cancel, socket, connID, wrappers.ConnPurposeClient, conf.PublicIPv4, conf.PublicIPv6)
	obfs2, connOpts, err := obfuscated2.ParseObfuscated2ClientFrame(conf.Secret, frame)
	if err != nil {
		return nil, nil, errors.Annotate(err, "Cannot parse obfuscated frame")
	}
	connOpts.ConnectionProto = mtproto.ConnectionProtocolAny
	connOpts.ClientAddr = conn.RemoteAddr()

	conn = wrappers.NewStreamCipher(conn, obfs2.Encryptor, obfs2.Decryptor)

	conn.Logger().Infow("Client connection initialized")

	return conn, connOpts, nil
}
