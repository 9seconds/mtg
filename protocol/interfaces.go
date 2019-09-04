package protocol

import (
	"github.com/9seconds/mtg/conntypes"
	"github.com/9seconds/mtg/telegram"
	"github.com/9seconds/mtg/wrappers"
)

type ClientProtocol interface {
	Handshake(wrappers.StreamReadWriteCloser) (wrappers.StreamReadWriteCloser, error)
	GetConnectionType() conntypes.ConnectionType
	GetConnectionProtocol() conntypes.ConnectionProtocol
	GetDC() conntypes.DC
}

type ClientProtocolMaker func() ClientProtocol

type TelegramProtocol interface {
	Handshake(*TelegramRequest) (wrappers.Wrap, error)
}

type TelegramProtocolMaker func(telegram.Telegram) TelegramProtocol
