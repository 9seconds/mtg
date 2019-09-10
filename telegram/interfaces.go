package telegram

import (
	"context"

	"github.com/9seconds/mtg/conntypes"
	"github.com/9seconds/mtg/wrappers"
)

type Telegram interface {
	Dial(context.Context,
		context.CancelFunc,
		conntypes.DC,
		conntypes.ConnectionProtocol) (wrappers.StreamReadWriteCloser, error)
	Secret() []byte
}
