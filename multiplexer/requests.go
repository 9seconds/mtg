package multiplexer

import "github.com/9seconds/mtg/mtproto"

type Request struct {
	data     []byte
	protocol mtproto.ConnectionProtocol
	dc       int16
	connID   ConnectionID
}
