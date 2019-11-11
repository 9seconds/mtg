package hub

import "github.com/9seconds/mtg/protocol"

type Interface interface {
	Register(*protocol.TelegramRequest) (*ProxyConn, error)
}
