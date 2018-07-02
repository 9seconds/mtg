package client

import (
	"net"

	"github.com/9seconds/mtg/config"
	"github.com/9seconds/mtg/mtproto"
	"github.com/9seconds/mtg/wrappers"
)

// Init has to initialize client connection based on given config.
type Init func(net.Conn, *config.Config) (*mtproto.ConnectionOpts, wrappers.ReadWriteCloserWithAddr, error)
