package faketls

import (
	"github.com/9seconds/mtg/conntypes"
	"github.com/9seconds/mtg/protocol"
)

func TelegramProtocol(req *protocol.TelegramRequest) (conntypes.StreamReadWriteCloser, error) {
	return nil, nil
}
