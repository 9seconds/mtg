package client

import (
	"context"
	"net"

	"github.com/9seconds/mtg/config"
	"github.com/9seconds/mtg/mtproto"
	"github.com/9seconds/mtg/wrappers"
)

// Init defines common method for initializing client connections.
type Init func(context.Context, context.CancelFunc, net.Conn, string,
	*config.Config) (wrappers.Wrap, *mtproto.ConnectionOpts, error)
