package telegram

import "mtg/conntypes"

type Telegram interface {
	Dial(conntypes.DC, conntypes.ConnectionProtocol) (conntypes.StreamReadWriteCloser, error)
	Secret() []byte
}
