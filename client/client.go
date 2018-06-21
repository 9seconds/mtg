package client

import (
	"io"
	"net"

	"github.com/9seconds/mtg/config"
	"github.com/9seconds/mtg/mtproto"
)

// Init has to initialize client connection based on given config.
type Init func(net.Conn, *config.Config) (*mtproto.ConnectionOpts, io.ReadWriteCloser, error)
