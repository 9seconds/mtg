package client

import (
	"io"
	"net"
	"time"

	"github.com/juju/errors"

	"github.com/9seconds/mtg/config"
	"github.com/9seconds/mtg/mtproto"
	"github.com/9seconds/mtg/obfuscated2"
	"github.com/9seconds/mtg/wrappers"
)

const (
	handshakeTimeout = 10 * time.Second
)

// DirectInit initializes client to access Telegram bypassing middleproxies.
func DirectInit(conn net.Conn, conf *config.Config) (*mtproto.ConnectionOpts, io.ReadWriteCloser, error) {
	if err := config.SetSocketOptions(conn); err != nil {
		return nil, nil, errors.Annotate(err, "Cannot set socket options")
	}

	conn.SetReadDeadline(time.Now().Add(handshakeTimeout)) // nolint: errcheck
	frame, err := obfuscated2.ExtractFrame(conn)
	conn.SetReadDeadline(time.Time{}) // nolint: errcheck
	if err != nil {
		return nil, nil, errors.Annotate(err, "Cannot extract frame")
	}
	defer obfuscated2.ReturnFrame(frame)

	obfs2, connOpts, err := obfuscated2.ParseObfuscated2ClientFrame(conf.Secret, frame)
	if err != nil {
		return nil, nil, errors.Annotate(err, "Cannot parse obfuscated frame")
	}

	socket := wrappers.NewStreamCipherRWC(conn, obfs2.Encryptor, obfs2.Decryptor)

	return connOpts, socket, nil
}
