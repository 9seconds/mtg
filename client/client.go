package client

import (
	"net"

	"mtg/config"
	"mtg/mtproto"
	"mtg/wrappers"
)

// Init defines common method for initializing client connections.
type Init func(net.Conn, string, *config.Config) (wrappers.Wrap, *mtproto.ConnectionOpts, error)
