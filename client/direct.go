package client

import (
	"io"
	"net"

	"github.com/juju/errors"

	"github.com/9seconds/mtg/config"
	"github.com/9seconds/mtg/mtproto"
	"github.com/9seconds/mtg/obfuscated2"
	"github.com/9seconds/mtg/wrappers"
)

// DirectInit initializes client to access Telegram bypassing middleproxies.
func DirectInit(conn net.Conn, conf *config.Config) (*mtproto.ConnectionOpts, io.ReadWriteCloser, error) {
	socket := wrappers.NewTimeoutRWC(conn, conf.TimeoutRead, conf.TimeoutWrite)
	frame, err := obfuscated2.ExtractFrame(socket)
	if err != nil {
		return nil, nil, errors.Annotate(err, "Cannot extract frame")
	}
	defer obfuscated2.ReturnFrame(frame)

	obfs2, connOpts, err := obfuscated2.ParseObfuscated2ClientFrame(conf.Secret, frame)
	if err != nil {
		return nil, nil, errors.Annotate(err, "Cannot parse obfuscated frame")
	}

	socket = wrappers.NewStreamCipherRWC(socket, obfs2.Encryptor, obfs2.Decryptor)

	return connOpts, socket, nil
}
