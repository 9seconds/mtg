package client

import (
	"net"

	"github.com/9seconds/mtg/config"
	"github.com/9seconds/mtg/mtproto"
	"github.com/9seconds/mtg/wrappers"
)

type Init func(net.Conn, string, *config.Config) (wrappers.Wrap, *mtproto.ConnectionOpts, error)
