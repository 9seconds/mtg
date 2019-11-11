package hub

import "mtg/protocol"

type Interface interface {
	Register(*protocol.TelegramRequest) (*ProxyConn, error)
}
