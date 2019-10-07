package telegram

import  "github.com/9seconds/mtg/conntypes"

type Telegram interface {
	Dial(conntypes.DC, conntypes.ConnectionProtocol) (conntypes.StreamReadWriteCloser, error)
	Secret() []byte
}
