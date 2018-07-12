package telegram

import (
	"github.com/9seconds/mtg/mtproto"
	"github.com/9seconds/mtg/wrappers"
)

// Telegram is an interface for different Telegram work modes.
type Telegram interface {
	ProxyDial(*mtproto.ConnectionOpts) (wrappers.StreamReadWriteCloser, error)
	ProxyInit(*mtproto.ConnectionOpts, wrappers.StreamReadWriteCloser) (wrappers.Wrap, error)
}

type TelegramMiddleDialer interface {
	Dial(int16, mtproto.ConnectionProtocol) (wrappers.PacketReadWriteCloser, error)
}
