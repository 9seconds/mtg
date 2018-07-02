package client

import (
	"net"
	"time"

	"github.com/juju/errors"

	"github.com/9seconds/mtg/config"
	"github.com/9seconds/mtg/mtproto"
	"github.com/9seconds/mtg/obfuscated2"
	"github.com/9seconds/mtg/wrappers"
)

const handshakeTimeout = 10 * time.Second

// DirectInit initializes client to access Telegram bypassing middleproxies.
func DirectInit(conn net.Conn, conf *config.Config) (*mtproto.ConnectionOpts, wrappers.ReadWriteCloserWithAddr, error) {
	if err := config.SetSocketOptions(conn); err != nil {
		return nil, nil, errors.Annotate(err, "Cannot set socket options")
	}

	conn.SetReadDeadline(time.Now().Add(handshakeTimeout)) // nolint: errcheck
	frame, err := obfuscated2.ExtractFrame(conn)
	conn.SetReadDeadline(time.Time{}) // nolint: errcheck
	if err != nil {
		return nil, nil, errors.Annotate(err, "Cannot extract frame")
	}

	obfs2, connOpts, err := obfuscated2.ParseObfuscated2ClientFrame(conf.Secret, frame)
	if err != nil {
		return nil, nil, errors.Annotate(err, "Cannot parse obfuscated frame")
	}
	connOpts.ConnectionProto = mtproto.ConnectionProtocolAny

	socket := wrappers.NewTimeoutRWC(conn)
	socket = wrappers.NewStreamCipherRWC(socket, obfs2.Encryptor, obfs2.Decryptor)

	return connOpts, socket, nil
}
