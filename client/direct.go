package client

import (
	"io"
	"net"

	"github.com/juju/errors"

	"github.com/9seconds/mtg/config"
	"github.com/9seconds/mtg/obfuscated2"
	"github.com/9seconds/mtg/wrappers"
)

func DirectInit(conn net.Conn, conf *config.Config) (int16, io.ReadWriteCloser, error) {
	socket := wrappers.NewTimeoutRWC(conn, conf.TimeoutRead, conf.TimeoutWrite)
	frame, err := obfuscated2.ExtractFrame(socket)
	if err != nil {
		return 0, nil, errors.Annotate(err, "Cannot extract frame")
	}

	obfs2, dc, err := obfuscated2.ParseObfuscated2ClientFrame(conf.Secret, frame)
	if err != nil {
		return 0, nil, errors.Annotate(err, "Cannot parse obfuscated frame")
	}

	socket = wrappers.NewStreamCipherRWC(socket, obfs2.Encryptor, obfs2.Decryptor)

	return dc, socket, nil
}
