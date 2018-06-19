package client

import (
	"io"
	"net"

	"github.com/9seconds/mtg/config"
)

// Init has to initialize client connection based on given config.
type Init func(net.Conn, *config.Config) (int16, io.ReadWriteCloser, error)
