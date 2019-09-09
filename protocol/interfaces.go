package protocol

import (
	"github.com/9seconds/mtg/conntypes"
	"github.com/9seconds/mtg/wrappers"
)

type ClientProtocol interface {
	Handshake(wrappers.StreamReadWriteCloser) (wrappers.StreamReadWriteCloser, error)
	ConnectionType() conntypes.ConnectionType
	ConnectionProtocol() conntypes.ConnectionProtocol
	DC() conntypes.DC
}

type TelegramProtocol interface {
	Handshake(*TelegramRequest) (wrappers.Wrap, error)
}

type ClientProtocolMaker func() ClientProtocol
type TelegramProtocolMaker func() TelegramProtocol
